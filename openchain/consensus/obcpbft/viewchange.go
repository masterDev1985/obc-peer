/*
Licensed to the Apache Software Foundation (ASF) under one
or more contributor license agreements.  See the NOTICE file
distributed with this work for additional information
regarding copyright ownership.  The ASF licenses this file
to you under the Apache License, Version 2.0 (the
"License"); you may not use this file except in compliance
with the License.  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied.  See the License for the
specific language governing permissions and limitations
under the License.
*/

package obcpbft

import "reflect"

func (instance *pbftCore) correctViewChange(vc *ViewChange) bool {
	for _, p := range append(vc.Pset, vc.Qset...) {
		if !(p.View < vc.View && p.SequenceNumber > vc.H && p.SequenceNumber <= vc.H+instance.L) {
			logger.Debug("invalid p entry in view-change: vc(v:%d h:%d) p(v:%d n:%d)",
				vc.View, vc.H, p.View, p.SequenceNumber)
			return false
		}
	}

	for _, c := range vc.Cset {
		// PBFT: the paper says c.n > vc.h
		if !(c.SequenceNumber >= vc.H && c.SequenceNumber <= vc.H+instance.L) {
			logger.Debug("invalid c entry in view-change: vc(v:%d h:%d) c(n:%d)",
				vc.View, vc.H, c.SequenceNumber)
			return false
		}
	}

	return true
}

func (instance *pbftCore) sendViewChange() error {
	instance.stopTimer()

	delete(instance.newViewStore, instance.view)
	instance.view++
	instance.activeView = false

	// P set: requests that have prepared here
	//
	// "<n,d,v> has a prepared certificate, and no request
	// prepared in a later view with the same number"

	for idx, cert := range instance.certStore {
		if cert.prePrepare == nil {
			continue
		}

		digest := cert.prePrepare.RequestDigest
		if !instance.prepared(digest, idx.v, idx.n) {
			continue
		}

		if p, ok := instance.pset[idx.n]; ok && p.View > idx.v {
			continue
		}

		instance.pset[idx.n] = &ViewChange_PQ{
			SequenceNumber: idx.n,
			Digest:         digest,
			View:           idx.v,
		}
	}

	// Q set: requests that have pre-prepared here (pre-prepare or
	// prepare sent)
	//
	// "<n,d,v>: requests that pre-prepared here, and did not
	// pre-prepare in a later view with the same number"

	for idx, cert := range instance.certStore {
		if cert.prePrepare == nil {
			continue
		}

		digest := cert.prePrepare.RequestDigest
		if !instance.prePrepared(digest, idx.v, idx.n) {
			continue
		}

		qi := qidx{digest, idx.n}
		if q, ok := instance.qset[qi]; ok && q.View > idx.v {
			continue
		}

		instance.qset[qi] = &ViewChange_PQ{
			SequenceNumber: idx.n,
			Digest:         digest,
			View:           idx.v,
		}
	}

	// clear old messages
	for idx := range instance.certStore {
		if idx.v < instance.view {
			delete(instance.certStore, idx)
		}
	}
	for idx := range instance.viewChangeStore {
		if idx.v < instance.view {
			delete(instance.viewChangeStore, idx)
		}
	}

	vc := &ViewChange{
		View:      instance.view,
		H:         instance.h,
		ReplicaId: instance.id,
	}

	for n, state := range instance.chkpts {
		vc.Cset = append(vc.Cset, &ViewChange_C{
			SequenceNumber: n,
			Digest:         state,
		})
	}

	for _, p := range instance.pset {
		vc.Pset = append(vc.Pset, p)
	}

	for _, q := range instance.qset {
		vc.Qset = append(vc.Qset, q)
	}

	instance.sign(vc)

	logger.Info("Replica %d sending view-change, v:%d, h:%d, |C|:%d, |P|:%d, |Q|:%d",
		instance.id, vc.View, vc.H, len(vc.Cset), len(vc.Pset), len(vc.Qset))

	return instance.innerBroadcast(&Message{&Message_ViewChange{vc}}, true)
}

func (instance *pbftCore) recvViewChange(vc *ViewChange) error {
	logger.Info("Replica %d received view-change from replica %d, v:%d, h:%d, |C|:%d, |P|:%d, |Q|:%d",
		instance.id, vc.ReplicaId, vc.View, vc.H, len(vc.Cset), len(vc.Pset), len(vc.Qset))

	if err := instance.verify(vc); err != nil {
		logger.Warning("incorrect signature in view-change message: %s", err)
		return nil
	}

	if !(vc.View >= instance.view && instance.correctViewChange(vc) && instance.viewChangeStore[vcidx{vc.View, vc.ReplicaId}] == nil) {
		logger.Warning("View-change message incorrect")
		return nil
	}

	instance.viewChangeStore[vcidx{vc.View, vc.ReplicaId}] = vc

	// PBFT TOCS 4.5.1 Liveness: "if a replica receives a set of
	// f+1 valid VIEW-CHANGE messages from other replicas for
	// views greater than its current view, it sends a VIEW-CHANGE
	// message for the smallest view in the set, even if its timer
	// has not expired"
	replicas := make(map[uint64]bool)
	minView := uint64(0)
	for idx := range instance.viewChangeStore {
		if idx.v <= instance.view {
			continue
		}

		replicas[idx.id] = true
		if minView == 0 || idx.v < minView {
			minView = idx.v
		}
	}
	if len(replicas) >= instance.f+1 {
		logger.Info("Replica %d received f+1 view-change messages, triggering view-change to view %d",
			instance.id, minView)
		// subtract one, because sendViewChange() increments
		instance.view = minView - 1
		return instance.sendViewChange()
	}

	quorum := 0
	for idx := range instance.viewChangeStore {
		if idx.v == instance.view {
			quorum++
		}
	}
	if vc.View == instance.view && quorum == 2*instance.f+1 {
		instance.startTimer(instance.lastNewViewTimeout)
		instance.lastNewViewTimeout = 2 * instance.lastNewViewTimeout
	}

	if instance.primary(instance.view) == instance.id {
		return instance.sendNewView()
	}

	return instance.processNewView()
}

func (instance *pbftCore) sendNewView() (err error) {
	if _, ok := instance.newViewStore[instance.view]; ok {
		return
	}

	vset := instance.getViewChanges()

	cp, ok := instance.selectInitialCheckpoint(vset)
	if !ok {
		return
	}

	msgList := instance.assignSequenceNumbers(vset, cp)
	if msgList == nil {
		return
	}

	nv := &NewView{
		View:      instance.view,
		Vset:      vset,
		Xset:      msgList,
		ReplicaId: instance.id,
	}

	logger.Info("New primary %d sending new-view, v:%d, X:%+v",
		instance.id, nv.View, nv.Xset)

	err = instance.innerBroadcast(&Message{&Message_NewView{nv}}, false)
	if err != nil {
		return err
	}
	instance.newViewStore[instance.view] = nv
	return instance.processNewView()
}

func (instance *pbftCore) recvNewView(nv *NewView) error {
	logger.Info("Replica %d received new-view %d",
		instance.id, nv.View)

	if !(nv.View > 0 && nv.View >= instance.view && instance.primary(nv.View) == nv.ReplicaId && instance.newViewStore[nv.View] == nil) {
		logger.Info("Replica %d rejecting invalid new-view from %d, v:%d",
			instance.id, nv.ReplicaId, nv.View)
		return nil
	}

	for _, vc := range nv.Vset {
		if err := instance.verify(vc); err != nil {
			logger.Warning("incorrect view-change signature in new-view message: %s", err)
			return nil
		}
	}

	instance.newViewStore[nv.View] = nv
	return instance.processNewView()
}

func (instance *pbftCore) processNewView() error {
	nv, ok := instance.newViewStore[instance.view]
	if !ok {
		return nil
	}

	if instance.activeView {
		logger.Info("Replica %d ignoring new-view from %d, v:%d: we are active in view %d",
			instance.id, nv.ReplicaId, nv.View, instance.view)
		return nil
	}

	cp, ok := instance.selectInitialCheckpoint(nv.Vset)
	if !ok {
		logger.Warning("could not determine initial checkpoint: %+v",
			instance.viewChangeStore)
		return instance.sendViewChange()
	}

	msgList := instance.assignSequenceNumbers(nv.Vset, cp)
	if msgList == nil {
		logger.Warning("could not assign sequence numbers: %+v",
			instance.viewChangeStore)
		return instance.sendViewChange()
	}

	if !(len(msgList) == 0 && len(nv.Xset) == 0) && !reflect.DeepEqual(msgList, nv.Xset) {
		logger.Warning("failed to verify new-view Xset: computed %+v, received %+v",
			msgList, nv.Xset)
		return instance.sendViewChange()
	}

	if instance.h < cp {
		logger.Warning("missing base checkpoint %d", cp)
		// XXX fetch checkpoint
		return nil
	}

	for n, d := range nv.Xset {
		// PBFT: why should we use "h ≥ min{n | ∃d : (<n,d> ∈ X)}"?
		// "h ≥ min{n | ∃d : (<n,d> ∈ X)} ∧ ∀<n,d> ∈ X : (n ≤ h ∨ ∃m ∈ in : (D(m) = d))"
		if n <= instance.h {
			continue
		} else {
			if d == "" {
				// NULL request; skip
				continue
			}

			if _, ok := instance.reqStore[d]; !ok {
				logger.Warning("missing assigned, non-checkpointed request %s",
					d)
				// XXX fetch request
				return nil
			}
		}
	}

	logger.Info("Replica %d accepting new-view to view %d", instance.id, instance.view)

	instance.activeView = true
	delete(instance.newViewStore, instance.view-1)

	for n, d := range nv.Xset {
		preprep := &PrePrepare{
			View:           instance.view,
			SequenceNumber: n,
			RequestDigest:  d,
			ReplicaId:      instance.id,
		}
		cert := instance.getCert(instance.view, n)
		cert.prePrepare = preprep
		if n > instance.seqNo {
			instance.seqNo = n
		}
	}

	if instance.primary(instance.view) != instance.id {
		for n, d := range nv.Xset {
			prep := &Prepare{
				View:           instance.view,
				SequenceNumber: n,
				RequestDigest:  d,
				ReplicaId:      instance.id,
			}
			cert := instance.getCert(instance.view, n)
			cert.prepare = append(cert.prepare, prep)
			cert.sentPrepare = true
			instance.innerBroadcast(&Message{&Message_Prepare{prep}}, true)
		}
	} else {
	outer:
		for d, req := range instance.outstandingReqs {
			for _, cert := range instance.certStore {
				if cert.prePrepare != nil && cert.prePrepare.RequestDigest == d {
					continue outer
				}
			}

			// This is a request that has not been pre-prepared yet
			// Trigger request processing again.
			instance.recvRequest(req)
		}
	}

	instance.consumer.viewChange(instance.view)

	return nil
}

func (instance *pbftCore) getViewChanges() (vset []*ViewChange) {
	for _, vc := range instance.viewChangeStore {
		vset = append(vset, vc)
	}

	return
}

func (instance *pbftCore) selectInitialCheckpoint(vset []*ViewChange) (checkpoint uint64, ok bool) {
	checkpoints := make(map[ViewChange_C][]*ViewChange)
	for _, vc := range vset {
		for _, c := range vc.Cset {
			checkpoints[*c] = append(checkpoints[*c], vc)
		}
	}

	if len(checkpoints) == 0 {
		logger.Debug("no checkpoints to select from: %d %s",
			len(instance.viewChangeStore), checkpoints)
		return
	}

	for idx, vcList := range checkpoints {
		// need weak certificate for the checkpoint
		if len(vcList) <= instance.f { // type casting necessary to match types
			logger.Debug("no weak certificate for n:%d",
				idx.SequenceNumber)
			continue
		}

		quorum := 0
		for _, vc := range vcList {
			if vc.H <= idx.SequenceNumber {
				quorum++
			}
		}

		if quorum <= 2*instance.f {
			logger.Debug("no quorum for n:%d",
				idx.SequenceNumber)
			continue
		}

		if checkpoint <= idx.SequenceNumber {
			checkpoint = idx.SequenceNumber
			ok = true
		}
	}

	return
}

func (instance *pbftCore) assignSequenceNumbers(vset []*ViewChange, h uint64) (msgList map[uint64]string) {
	msgList = make(map[uint64]string)

	maxN := h

	// "for all n such that h < n <= h + L"
nLoop:
	for n := h + 1; n <= h+instance.L; n++ {
		// "∃m ∈ S..."
		for _, m := range vset {
			// "...with <n,d,v> ∈ m.P"
			for _, em := range m.Pset {
				quorum := 0
				// "A1. ∃2f+1 messages m' ∈ S"
			mpLoop:
				for _, mp := range vset {
					if mp.H >= n {
						continue
					}
					// "∀<n,d',v'> ∈ m'.P"
					for _, emp := range mp.Pset {
						if n == emp.SequenceNumber && !(emp.View < em.View || (emp.View == em.View && emp.Digest == em.Digest)) {
							continue mpLoop
						}
					}
					quorum++
				}

				if quorum < 2*instance.f+1 {
					continue
				}

				quorum = 0
				// "A2. ∃f+1 messages m' ∈ S"
				for _, mp := range vset {
					// "∃<n,d',v'> ∈ m'.Q"
					for _, emp := range mp.Qset {
						if n == emp.SequenceNumber && emp.View >= em.View && emp.Digest == em.Digest {
							quorum++
						}
					}
				}

				if quorum < instance.f+1 {
					continue
				}

				// "then select the request with digest d for number n"
				msgList[n] = em.Digest
				maxN = n

				continue nLoop
			}
		}

		quorum := 0
		// "else if ∃2f+1 messages m ∈ S"
	nullLoop:
		for _, m := range vset {
			// "m.P has no entry"
			for _, em := range m.Pset {
				if em.SequenceNumber == n {
					continue nullLoop
				}
			}
			quorum++
		}

		if quorum >= 2*instance.f+1 {
			// "then select the null request for number n"
			msgList[n] = ""

			continue nLoop
		}

		return nil
	}

	// prune top null requests
	for n, msg := range msgList {
		if n > maxN && msg == "" {
			delete(msgList, n)
		}
	}

	return
}
