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

package ledger

import (
	"bytes"
	"strconv"
	"testing"

	"github.com/openblockchain/obc-peer/openchain/ledger/statemgmt"
	"github.com/openblockchain/obc-peer/openchain/ledger/testutil"
	"github.com/openblockchain/obc-peer/protos"
)

func TestLedgerCommit(t *testing.T) {
	ledgerTestWrapper := createFreshDBAndTestLedgerWrapper(t)
	ledger := ledgerTestWrapper.ledger
	ledger.BeginTxBatch(1)
	ledger.TxBegin("txUuid")
	ledger.SetState("chaincode1", "key1", []byte("value1"))
	ledger.SetState("chaincode2", "key2", []byte("value2"))
	ledger.SetState("chaincode3", "key3", []byte("value3"))
	ledger.TxFinished("txUuid", true)
	transaction, _ := buildTestTx()
	ledger.CommitTxBatch(1, []*protos.Transaction{transaction}, []byte("proof"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode1", "key1", false), []byte("value1"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode1", "key1", true), []byte("value1"))
}

func TestLedgerRollback(t *testing.T) {
	ledgerTestWrapper := createFreshDBAndTestLedgerWrapper(t)
	ledger := ledgerTestWrapper.ledger
	ledger.BeginTxBatch(1)
	ledger.TxBegin("txUuid")
	ledger.SetState("chaincode1", "key1", []byte("value1"))
	ledger.SetState("chaincode2", "key2", []byte("value2"))
	ledger.SetState("chaincode3", "key3", []byte("value3"))
	ledger.TxFinished("txUuid", true)
	ledger.RollbackTxBatch(1)
	testutil.AssertNil(t, ledgerTestWrapper.GetState("chaincode1", "key1", false))
}

func TestLedgerRollbackWithHash(t *testing.T) {
	ledgerTestWrapper := createFreshDBAndTestLedgerWrapper(t)
	ledger := ledgerTestWrapper.ledger

	ledger.BeginTxBatch(0)
	ledger.TxBegin("txUuid")
	ledger.SetState("chaincode0", "key1", []byte("value1"))
	ledger.SetState("chaincode0", "key2", []byte("value2"))
	ledger.SetState("chaincode0", "key3", []byte("value3"))
	ledger.TxFinished("txUuid", true)
	ledger.RollbackTxBatch(0)

	hash0 := ledgerTestWrapper.GetTempStateHash()

	ledger.BeginTxBatch(1)
	ledger.TxBegin("txUuid")
	ledger.SetState("chaincode1", "key1", []byte("value1"))
	ledger.SetState("chaincode2", "key2", []byte("value2"))
	ledger.SetState("chaincode3", "key3", []byte("value3"))
	ledger.TxFinished("txUuid", true)

	hash1 := ledgerTestWrapper.GetTempStateHash()
	testutil.AssertNotEquals(t, hash1, hash0)

	ledger.RollbackTxBatch(1)
	hash1 = ledgerTestWrapper.GetTempStateHash()
	testutil.AssertEquals(t, hash1, hash0)
	testutil.AssertNil(t, ledgerTestWrapper.GetState("chaincode1", "key1", false))
}

func TestLedgerDifferentID(t *testing.T) {
	ledgerTestWrapper := createFreshDBAndTestLedgerWrapper(t)
	ledger := ledgerTestWrapper.ledger
	ledger.BeginTxBatch(1)
	ledger.TxBegin("txUuid")
	ledger.SetState("chaincode1", "key1", []byte("value1"))
	ledger.SetState("chaincode2", "key2", []byte("value2"))
	ledger.SetState("chaincode3", "key3", []byte("value3"))
	ledger.TxFinished("txUuid", true)
	transaction, _ := buildTestTx()
	err := ledger.CommitTxBatch(2, []*protos.Transaction{transaction}, []byte("prrof"))
	testutil.AssertError(t, err, "ledger should throw error for wrong batch ID")
}

func TestLedgerGetTempStateHashWithTxDeltaStateHashes(t *testing.T) {
	ledgerTestWrapper := createFreshDBAndTestLedgerWrapper(t)
	ledger := ledgerTestWrapper.ledger
	ledger.BeginTxBatch(1)
	ledger.TxBegin("txUuid1")
	ledger.SetState("chaincode1", "key1", []byte("value1"))
	ledger.TxFinished("txUuid1", true)

	ledger.TxBegin("txUuid2")
	ledger.SetState("chaincode2", "key2", []byte("value2"))
	ledger.TxFinished("txUuid2", true)

	ledger.TxBegin("txUuid3")
	ledger.TxFinished("txUuid3", true)

	ledger.TxBegin("txUuid4")
	ledger.SetState("chaincode4", "key4", []byte("value4"))
	ledger.TxFinished("txUuid4", false)

	_, txDeltaHashes, _ := ledger.GetTempStateHashWithTxDeltaStateHashes()
	testutil.AssertEquals(t, testutil.ComputeCryptoHash([]byte("chaincode1key1value1")), txDeltaHashes["txUuid1"])
	testutil.AssertEquals(t, testutil.ComputeCryptoHash([]byte("chaincode2key2value2")), txDeltaHashes["txUuid2"])
	testutil.AssertNil(t, txDeltaHashes["txUuid3"])
	_, ok := txDeltaHashes["txUuid4"]
	if ok {
		t.Fatalf("Entry for a failed Tx should not be present in txDeltaHashes map")
	}
	ledger.CommitTxBatch(1, []*protos.Transaction{}, []byte("proof"))

	ledger.BeginTxBatch(2)
	ledger.TxBegin("txUuid1")
	ledger.SetState("chaincode1", "key1", []byte("value1"))
	ledger.TxFinished("txUuid1", true)
	_, txDeltaHashes, _ = ledger.GetTempStateHashWithTxDeltaStateHashes()
	if len(txDeltaHashes) != 1 {
		t.Fatalf("Entries in txDeltaHashes map should only be from current batch")
	}
}

func TestLedgerStateSnapshot(t *testing.T) {
	ledgerTestWrapper := createFreshDBAndTestLedgerWrapper(t)
	ledger := ledgerTestWrapper.ledger
	ledger.BeginTxBatch(1)
	ledger.TxBegin("txUuid")
	ledger.SetState("chaincode1", "key1", []byte("value1"))
	ledger.SetState("chaincode2", "key2", []byte("value2"))
	ledger.SetState("chaincode3", "key3", []byte("value3"))
	ledger.TxFinished("txUuid", true)
	transaction, _ := buildTestTx()
	ledger.CommitTxBatch(1, []*protos.Transaction{transaction}, []byte("proof"))

	snapshot, err := ledger.GetStateSnapshot()

	if err != nil {
		t.Fatalf("Error fetching snapshot %s", err)
	}
	defer snapshot.Release()

	// Modify keys to ensure they do not impact the snapshot
	ledger.BeginTxBatch(2)
	ledger.TxBegin("txUuid")
	ledger.DeleteState("chaincode1", "key1")
	ledger.SetState("chaincode4", "key4", []byte("value4"))
	ledger.SetState("chaincode5", "key5", []byte("value5"))
	ledger.SetState("chaincode6", "key6", []byte("value6"))
	ledger.TxFinished("txUuid", true)
	transaction, _ = buildTestTx()
	ledger.CommitTxBatch(2, []*protos.Transaction{transaction}, []byte("proof"))

	var count = 0
	for snapshot.Next() {
		k, v := snapshot.GetRawKeyValue()
		t.Logf("Key %v, Val %v", k, v)
		count++
	}
	if count != 3 {
		t.Fatalf("Expected 3 keys, but got %d", count)
	}

	if snapshot.GetBlockNumber() != 1 {
		t.Fatalf("Expected blocknumber to be 1, but got %d", snapshot.GetBlockNumber())
	}

}

func TestLedgerPutRawBlock(t *testing.T) {
	ledgerTestWrapper := createFreshDBAndTestLedgerWrapper(t)
	ledger := ledgerTestWrapper.ledger
	block := new(protos.Block)
	block.PreviousBlockHash = []byte("foo")
	block.StateHash = []byte("bar")
	ledger.PutRawBlock(block, 4)
	testutil.AssertEquals(t, ledgerTestWrapper.GetBlockByNumber(4), block)
}

func TestLedgerSetRawState(t *testing.T) {
	ledgerTestWrapper := createFreshDBAndTestLedgerWrapper(t)
	ledger := ledgerTestWrapper.ledger
	ledger.BeginTxBatch(1)
	ledger.TxBegin("txUuid1")
	ledger.SetState("chaincode1", "key1", []byte("value1"))
	ledger.SetState("chaincode2", "key2", []byte("value2"))
	ledger.SetState("chaincode3", "key3", []byte("value3"))
	ledger.TxFinished("txUuid1", true)
	transaction, _ := buildTestTx()
	ledger.CommitTxBatch(1, []*protos.Transaction{transaction}, []byte("proof"))

	// Ensure values are in the DB
	val := ledgerTestWrapper.GetState("chaincode1", "key1", true)
	if bytes.Compare(val, []byte("value1")) != 0 {
		t.Fatalf("Expected initial chaincode1 key1 to be %s, but got %s", []byte("value1"), val)
	}
	val = ledgerTestWrapper.GetState("chaincode2", "key2", true)
	if bytes.Compare(val, []byte("value2")) != 0 {
		t.Fatalf("Expected initial chaincode1 key2 to be %s, but got %s", []byte("value2"), val)
	}
	val = ledgerTestWrapper.GetState("chaincode3", "key3", true)
	if bytes.Compare(val, []byte("value3")) != 0 {
		t.Fatalf("Expected initial chaincode1 key3 to be %s, but got %s", []byte("value3"), val)
	}

	hash1, hash1Err := ledger.GetTempStateHash()
	if hash1Err != nil {
		t.Fatalf("Error getting hash1 %s", hash1Err)
	}

	snapshot, snapshotError := ledger.GetStateSnapshot()
	if snapshotError != nil {
		t.Fatalf("Error fetching snapshot %s", snapshotError)
	}
	defer snapshot.Release()

	// Delete keys
	ledger.BeginTxBatch(2)
	ledger.TxBegin("txUuid2")
	ledger.DeleteState("chaincode1", "key1")
	ledger.DeleteState("chaincode2", "key2")
	ledger.DeleteState("chaincode3", "key3")
	ledger.TxFinished("txUuid2", true)
	transaction, _ = buildTestTx()
	ledger.CommitTxBatch(2, []*protos.Transaction{transaction}, []byte("proof"))

	// ensure keys are deleted
	val = ledgerTestWrapper.GetState("chaincode1", "key1", true)
	if val != nil {
		t.Fatalf("Expected chaincode1 key1 to be nil, but got %s", val)
	}
	val = ledgerTestWrapper.GetState("chaincode2", "key2", true)
	if val != nil {
		t.Fatalf("Expected chaincode2 key2 to be nil, but got %s", val)
	}
	val = ledgerTestWrapper.GetState("chaincode3", "key3", true)
	if val != nil {
		t.Fatalf("Expected chaincode3 key3 to be nil, but got %s", val)
	}

	hash2, hash2Err := ledger.GetTempStateHash()
	if hash2Err != nil {
		t.Fatalf("Error getting hash2 %s", hash2Err)
	}

	if bytes.Compare(hash1, hash2) == 0 {
		t.Fatalf("Expected hashes to not match, but they both equal %s", hash1)
	}

	// put key/values from the snapshot back in the DB
	//var keys, values [][]byte
	delta := statemgmt.NewStateDelta()
	for i := 0; snapshot.Next(); i++ {
		k, v := snapshot.GetRawKeyValue()
		cID, keyID := statemgmt.DecodeCompositeKey(k)
		delta.Set(cID, keyID, v, nil)
	}

	ledgerTestWrapper.ApplyStateDelta(1, delta)
	ledgerTestWrapper.CommitStateDelta(1)

	// Ensure values are back in the DB
	val = ledgerTestWrapper.GetState("chaincode1", "key1", true)
	if bytes.Compare(val, []byte("value1")) != 0 {
		t.Fatalf("Expected chaincode1 key1 to be %s, but got %s", []byte("value1"), val)
	}
	val = ledgerTestWrapper.GetState("chaincode2", "key2", true)
	if bytes.Compare(val, []byte("value2")) != 0 {
		t.Fatalf("Expected chaincode1 key2 to be %s, but got %s", []byte("value2"), val)
	}
	val = ledgerTestWrapper.GetState("chaincode3", "key3", true)
	if bytes.Compare(val, []byte("value3")) != 0 {
		t.Fatalf("Expected chaincode1 key3 to be %s, but got %s", []byte("value3"), val)
	}

	hash3, hash3Err := ledger.GetTempStateHash()
	if hash3Err != nil {
		t.Fatalf("Error getting hash3 %s", hash3Err)
	}
	if bytes.Compare(hash1, hash3) != 0 {
		t.Fatalf("Expected hashes to be equal, but they are %s and %s", hash1, hash3)
	}
}

func TestDeleteAllStateKeysAndValues(t *testing.T) {
	ledgerTestWrapper := createFreshDBAndTestLedgerWrapper(t)
	ledger := ledgerTestWrapper.ledger
	ledger.BeginTxBatch(1)
	ledger.TxBegin("txUuid1")
	ledger.SetState("chaincode1", "key1", []byte("value1"))
	ledger.SetState("chaincode2", "key2", []byte("value2"))
	ledger.SetState("chaincode3", "key3", []byte("value3"))
	ledger.TxFinished("txUuid1", true)
	transaction, _ := buildTestTx()
	ledger.CommitTxBatch(1, []*protos.Transaction{transaction}, []byte("proof"))

	// Confirm values are present in state
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode1", "key1", true), []byte("value1"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode2", "key2", true), []byte("value2"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode3", "key3", true), []byte("value3"))

	// Delete all keys/values
	err := ledger.DeleteALLStateKeysAndValues()
	if err != nil {
		t.Fatalf("Error calling deleting all keys/values from state: %s", err)
	}

	// Confirm values are deleted
	testutil.AssertNil(t, ledgerTestWrapper.GetState("chaincode1", "key1", true))
	testutil.AssertNil(t, ledgerTestWrapper.GetState("chaincode2", "key2", true))
	testutil.AssertNil(t, ledgerTestWrapper.GetState("chaincode3", "key3", true))

	// Test that we can now store new stuff in the state
	ledger.BeginTxBatch(2)
	ledger.TxBegin("txUuid1")
	ledger.SetState("chaincode1", "key1", []byte("value1"))
	ledger.SetState("chaincode2", "key2", []byte("value2"))
	ledger.SetState("chaincode3", "key3", []byte("value3"))
	ledger.TxFinished("txUuid1", true)
	transaction, _ = buildTestTx()
	ledger.CommitTxBatch(2, []*protos.Transaction{transaction}, []byte("proof"))

	// Confirm values are present in state
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode1", "key1", true), []byte("value1"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode2", "key2", true), []byte("value2"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode3", "key3", true), []byte("value3"))
}

func TestVerifyChain(t *testing.T) {
	ledgerTestWrapper := createFreshDBAndTestLedgerWrapper(t)
	ledger := ledgerTestWrapper.ledger

	// Build a big blockchain
	for i := 0; i < 100; i++ {
		ledger.BeginTxBatch(i)
		ledger.TxBegin("txUuid" + strconv.Itoa(i))
		ledger.SetState("chaincode"+strconv.Itoa(i), "key"+strconv.Itoa(i), []byte("value"+strconv.Itoa(i)))
		ledger.TxFinished("txUuid"+strconv.Itoa(i), true)
		transaction, _ := buildTestTx()
		ledger.CommitTxBatch(i, []*protos.Transaction{transaction}, []byte("proof"))
	}

	// Verify the chain
	for lowBlock := uint64(0); lowBlock < ledger.GetBlockchainSize()-1; lowBlock++ {
		testutil.AssertEquals(t, ledgerTestWrapper.VerifyChain(ledger.GetBlockchainSize()-1, lowBlock), uint64(0))
	}
	for highBlock := ledger.GetBlockchainSize() - 1; highBlock > 0; highBlock-- {
		testutil.AssertEquals(t, ledgerTestWrapper.VerifyChain(highBlock, 0), uint64(0))
	}

	// Add bad blocks and test
	badBlock := protos.NewBlock(nil)
	badBlock.PreviousBlockHash = []byte("evil")
	for i := uint64(0); i < ledger.GetBlockchainSize(); i++ {
		goodBlock := ledgerTestWrapper.GetBlockByNumber(i)
		ledger.PutRawBlock(badBlock, i)
		for lowBlock := uint64(0); lowBlock < ledger.GetBlockchainSize()-1; lowBlock++ {
			if i >= lowBlock {
				expected := uint64(i + 1)
				if i == ledger.GetBlockchainSize()-1 {
					expected--
				}
				testutil.AssertEquals(t, ledgerTestWrapper.VerifyChain(ledger.GetBlockchainSize()-1, lowBlock), expected)
			} else {
				testutil.AssertEquals(t, ledgerTestWrapper.VerifyChain(ledger.GetBlockchainSize()-1, lowBlock), uint64(0))
			}
		}
		for highBlock := ledger.GetBlockchainSize() - 1; highBlock > 0; highBlock-- {
			if i <= highBlock {
				expected := uint64(i + 1)
				if i == highBlock {
					expected--
				}
				testutil.AssertEquals(t, ledgerTestWrapper.VerifyChain(highBlock, 0), expected)
			} else {
				testutil.AssertEquals(t, ledgerTestWrapper.VerifyChain(highBlock, 0), uint64(0))
			}
		}
		ledgerTestWrapper.PutRawBlock(goodBlock, i)
	}

	// Test edge cases
	_, err := ledger.VerifyChain(2, 10)
	testutil.AssertError(t, err, "Expected error as high block is less than low block")
	_, err = ledger.VerifyChain(2, 2)
	testutil.AssertError(t, err, "Expected error as high block is equal to low block")
	_, err = ledger.VerifyChain(0, 100)
	testutil.AssertError(t, err, "Expected error as high block is out of bounds")
}

func TestBlockNumberOutOfBoundsError(t *testing.T) {
	ledgerTestWrapper := createFreshDBAndTestLedgerWrapper(t)
	ledger := ledgerTestWrapper.ledger

	// Build a big blockchain
	for i := 0; i < 10; i++ {
		ledger.BeginTxBatch(i)
		ledger.TxBegin("txUuid" + strconv.Itoa(i))
		ledger.SetState("chaincode"+strconv.Itoa(i), "key"+strconv.Itoa(i), []byte("value"+strconv.Itoa(i)))
		ledger.TxFinished("txUuid"+strconv.Itoa(i), true)
		transaction, _ := buildTestTx()
		ledger.CommitTxBatch(i, []*protos.Transaction{transaction}, []byte("proof"))
	}

	ledgerTestWrapper.GetBlockByNumber(9)
	_, err := ledger.GetBlockByNumber(10)
	testutil.AssertEquals(t, err, ErrOutOfBounds)

	ledgerTestWrapper.GetStateDelta(9)
	_, err = ledger.GetStateDelta(10)
	testutil.AssertEquals(t, err, ErrOutOfBounds)

}

func TestRollBackwardsAndForwards(t *testing.T) {

	ledgerTestWrapper := createFreshDBAndTestLedgerWrapper(t)
	ledger := ledgerTestWrapper.ledger

	// Block 0
	ledger.BeginTxBatch(0)
	ledger.TxBegin("txUuid1")
	ledger.SetState("chaincode1", "key1", []byte("value1A"))
	ledger.SetState("chaincode2", "key2", []byte("value2A"))
	ledger.SetState("chaincode3", "key3", []byte("value3A"))
	ledger.TxFinished("txUuid1", true)
	transaction, _ := buildTestTx()
	ledger.CommitTxBatch(0, []*protos.Transaction{transaction}, []byte("proof"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode1", "key1", true), []byte("value1A"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode2", "key2", true), []byte("value2A"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode3", "key3", true), []byte("value3A"))

	// Block 1
	ledger.BeginTxBatch(1)
	ledger.TxBegin("txUuid1")
	ledger.SetState("chaincode1", "key1", []byte("value1B"))
	ledger.SetState("chaincode2", "key2", []byte("value2B"))
	ledger.SetState("chaincode3", "key3", []byte("value3B"))
	ledger.TxFinished("txUuid1", true)
	transaction, _ = buildTestTx()
	ledger.CommitTxBatch(1, []*protos.Transaction{transaction}, []byte("proof"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode1", "key1", true), []byte("value1B"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode2", "key2", true), []byte("value2B"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode3", "key3", true), []byte("value3B"))

	// Block 2
	ledger.BeginTxBatch(2)
	ledger.TxBegin("txUuid1")
	ledger.SetState("chaincode1", "key1", []byte("value1C"))
	ledger.SetState("chaincode2", "key2", []byte("value2C"))
	ledger.SetState("chaincode3", "key3", []byte("value3C"))
	ledger.SetState("chaincode4", "key4", []byte("value4C"))
	ledger.TxFinished("txUuid1", true)
	transaction, _ = buildTestTx()
	ledger.CommitTxBatch(2, []*protos.Transaction{transaction}, []byte("proof"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode1", "key1", true), []byte("value1C"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode2", "key2", true), []byte("value2C"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode3", "key3", true), []byte("value3C"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode4", "key4", true), []byte("value4C"))

	// Roll backwards once
	delta2 := ledgerTestWrapper.GetStateDelta(2)
	delta2.RollBackwards = true
	ledgerTestWrapper.ApplyStateDelta(1, delta2)
	ledgerTestWrapper.CommitStateDelta(1)
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode1", "key1", true), []byte("value1B"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode2", "key2", true), []byte("value2B"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode3", "key3", true), []byte("value3B"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode4", "key4", true), nil)

	// Now roll forwards once
	delta2.RollBackwards = false
	ledgerTestWrapper.ApplyStateDelta(2, delta2)
	ledgerTestWrapper.CommitStateDelta(2)
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode1", "key1", true), []byte("value1C"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode2", "key2", true), []byte("value2C"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode3", "key3", true), []byte("value3C"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode4", "key4", true), []byte("value4C"))

	// Now roll backwards twice
	delta2.RollBackwards = true
	delta1 := ledgerTestWrapper.GetStateDelta(1)
	ledgerTestWrapper.ApplyStateDelta(3, delta2)
	ledgerTestWrapper.CommitStateDelta(3)

	delta1.RollBackwards = true
	ledgerTestWrapper.ApplyStateDelta(4, delta1)
	ledgerTestWrapper.CommitStateDelta(4)
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode1", "key1", true), []byte("value1A"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode2", "key2", true), []byte("value2A"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode3", "key3", true), []byte("value3A"))

	// Now roll forwards twice
	delta2.RollBackwards = false
	delta1.RollBackwards = false
	ledgerTestWrapper.ApplyStateDelta(5, delta1)
	ledgerTestWrapper.CommitStateDelta(5)
	ledgerTestWrapper.ApplyStateDelta(6, delta2)
	ledgerTestWrapper.CommitStateDelta(6)
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode1", "key1", true), []byte("value1C"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode2", "key2", true), []byte("value2C"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode3", "key3", true), []byte("value3C"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode4", "key4", true), []byte("value4C"))
}

func TestInvalidOrderDelta(t *testing.T) {
	ledgerTestWrapper := createFreshDBAndTestLedgerWrapper(t)
	ledger := ledgerTestWrapper.ledger

	// Block 0
	ledger.BeginTxBatch(0)
	ledger.TxBegin("txUuid1")
	ledger.SetState("chaincode1", "key1", []byte("value1A"))
	ledger.SetState("chaincode2", "key2", []byte("value2A"))
	ledger.SetState("chaincode3", "key3", []byte("value3A"))
	ledger.TxFinished("txUuid1", true)
	transaction, _ := buildTestTx()
	ledger.CommitTxBatch(0, []*protos.Transaction{transaction}, []byte("proof"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode1", "key1", true), []byte("value1A"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode2", "key2", true), []byte("value2A"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode3", "key3", true), []byte("value3A"))

	// Block 1
	ledger.BeginTxBatch(1)
	ledger.TxBegin("txUuid1")
	ledger.SetState("chaincode1", "key1", []byte("value1B"))
	ledger.SetState("chaincode2", "key2", []byte("value2B"))
	ledger.SetState("chaincode3", "key3", []byte("value3B"))
	ledger.TxFinished("txUuid1", true)
	transaction, _ = buildTestTx()
	ledger.CommitTxBatch(1, []*protos.Transaction{transaction}, []byte("proof"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode1", "key1", true), []byte("value1B"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode2", "key2", true), []byte("value2B"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode3", "key3", true), []byte("value3B"))

	delta := ledgerTestWrapper.GetStateDelta(1)

	err := ledger.CommitStateDelta(1)
	testutil.AssertError(t, err, "Expected error commiting delta")

	err = ledger.RollbackTxBatch(1)
	testutil.AssertError(t, err, "Expected error rolling back delta")

	ledgerTestWrapper.ApplyStateDelta(2, delta)

	err = ledger.ApplyStateDelta(3, delta)
	testutil.AssertError(t, err, "Expected error applying delta")

	err = ledger.CommitStateDelta(3)
	testutil.AssertError(t, err, "Expected error applying delta")

	err = ledger.RollbackStateDelta(3)
	testutil.AssertError(t, err, "Expected error applying delta")

}

func TestApplyDeltaHash(t *testing.T) {

	ledgerTestWrapper := createFreshDBAndTestLedgerWrapper(t)
	ledger := ledgerTestWrapper.ledger

	// Block 0
	ledger.BeginTxBatch(0)
	ledger.TxBegin("txUuid1")
	ledger.SetState("chaincode1", "key1", []byte("value1A"))
	ledger.SetState("chaincode2", "key2", []byte("value2A"))
	ledger.SetState("chaincode3", "key3", []byte("value3A"))
	ledger.TxFinished("txUuid1", true)
	transaction, _ := buildTestTx()
	ledger.CommitTxBatch(0, []*protos.Transaction{transaction}, []byte("proof"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode1", "key1", true), []byte("value1A"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode2", "key2", true), []byte("value2A"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode3", "key3", true), []byte("value3A"))

	// Block 1
	ledger.BeginTxBatch(1)
	ledger.TxBegin("txUuid1")
	ledger.SetState("chaincode1", "key1", []byte("value1B"))
	ledger.SetState("chaincode2", "key2", []byte("value2B"))
	ledger.SetState("chaincode3", "key3", []byte("value3B"))
	ledger.TxFinished("txUuid1", true)
	transaction, _ = buildTestTx()
	ledger.CommitTxBatch(1, []*protos.Transaction{transaction}, []byte("proof"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode1", "key1", true), []byte("value1B"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode2", "key2", true), []byte("value2B"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode3", "key3", true), []byte("value3B"))

	// Block 2
	ledger.BeginTxBatch(2)
	ledger.TxBegin("txUuid1")
	ledger.SetState("chaincode1", "key1", []byte("value1C"))
	ledger.SetState("chaincode2", "key2", []byte("value2C"))
	ledger.SetState("chaincode3", "key3", []byte("value3C"))
	ledger.SetState("chaincode4", "key4", []byte("value4C"))
	ledger.TxFinished("txUuid1", true)
	transaction, _ = buildTestTx()
	ledger.CommitTxBatch(2, []*protos.Transaction{transaction}, []byte("proof"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode1", "key1", true), []byte("value1C"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode2", "key2", true), []byte("value2C"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode3", "key3", true), []byte("value3C"))
	testutil.AssertEquals(t, ledgerTestWrapper.GetState("chaincode4", "key4", true), []byte("value4C"))

	hash2 := ledgerTestWrapper.GetTempStateHash()

	// Roll backwards once
	delta2 := ledgerTestWrapper.GetStateDelta(2)
	delta2.RollBackwards = true
	ledgerTestWrapper.ApplyStateDelta(1, delta2)

	preHash1 := ledgerTestWrapper.GetTempStateHash()
	testutil.AssertNotEquals(t, preHash1, hash2)

	ledgerTestWrapper.CommitStateDelta(1)

	hash1 := ledgerTestWrapper.GetTempStateHash()
	testutil.AssertEquals(t, preHash1, hash1)
	testutil.AssertNotEquals(t, hash1, hash2)

	// Roll forwards once
	delta2.RollBackwards = false
	ledgerTestWrapper.ApplyStateDelta(2, delta2)
	preHash2 := ledgerTestWrapper.GetTempStateHash()
	testutil.AssertEquals(t, preHash2, hash2)
	ledgerTestWrapper.RollbackStateDelta(2)
	preHash2 = ledgerTestWrapper.GetTempStateHash()
	testutil.AssertEquals(t, preHash2, hash1)
	ledgerTestWrapper.ApplyStateDelta(3, delta2)
	preHash2 = ledgerTestWrapper.GetTempStateHash()
	testutil.AssertEquals(t, preHash2, hash2)
	ledgerTestWrapper.CommitStateDelta(3)
	preHash2 = ledgerTestWrapper.GetTempStateHash()
	testutil.AssertEquals(t, preHash2, hash2)

}

func TestPreviewTXBatchBlock(t *testing.T) {
	ledgerTestWrapper := createFreshDBAndTestLedgerWrapper(t)
	ledger := ledgerTestWrapper.ledger

	// Block 0
	ledger.BeginTxBatch(0)
	ledger.TxBegin("txUuid1")
	ledger.SetState("chaincode1", "key1", []byte("value1A"))
	ledger.SetState("chaincode2", "key2", []byte("value2A"))
	ledger.SetState("chaincode3", "key3", []byte("value3A"))
	ledger.TxFinished("txUuid1", true)
	transaction, _ := buildTestTx()

	previewBlock, err := ledger.GetTXBatchPreviewBlock(0, []*protos.Transaction{transaction}, []byte("proof"))
	testutil.AssertNoError(t, err, "Error fetching preview block.")

	ledger.CommitTxBatch(0, []*protos.Transaction{transaction}, []byte("proof"))
	commitedBlock := ledgerTestWrapper.GetBlockByNumber(0)

	previewBlockHash, err := previewBlock.GetHash()
	testutil.AssertNoError(t, err, "Error fetching preview block hash.")

	commitedBlockHash, err := commitedBlock.GetHash()
	testutil.AssertNoError(t, err, "Error fetching committed block hash.")

	testutil.AssertEquals(t, previewBlockHash, commitedBlockHash)
}

func TestGetTransactionByUUID(t *testing.T) {
	ledgerTestWrapper := createFreshDBAndTestLedgerWrapper(t)
	ledger := ledgerTestWrapper.ledger

	// Block 0
	ledger.BeginTxBatch(0)
	ledger.TxBegin("txUuid1")
	ledger.SetState("chaincode1", "key1", []byte("value1A"))
	ledger.SetState("chaincode2", "key2", []byte("value2A"))
	ledger.SetState("chaincode3", "key3", []byte("value3A"))
	ledger.TxFinished("txUuid1", true)
	transaction, uuid := buildTestTx()
	ledger.CommitTxBatch(0, []*protos.Transaction{transaction}, []byte("proof"))

	ledgerTransaction, err := ledger.GetTransactionByUUID(uuid)
	testutil.AssertNoError(t, err, "Error fetching transaction by UUID.")
	testutil.AssertEquals(t, transaction, ledgerTransaction)

	ledgerTransaction, err = ledger.GetTransactionByUUID("InvalidUUID")
	testutil.AssertEquals(t, err, ErrResourceNotFound)
	testutil.AssertNil(t, ledgerTransaction)
}
