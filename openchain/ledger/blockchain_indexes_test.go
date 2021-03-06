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
	"testing"

	"github.com/openblockchain/obc-peer/openchain/ledger/testutil"
	"github.com/openblockchain/obc-peer/protos"
)

func TestIndexes_GetBlockByBlockNumber(t *testing.T) {
	testDBWrapper.CreateFreshDB(t)
	testBlockchainWrapper := newTestBlockchainWrapper(t)
	blocks, _ := testBlockchainWrapper.populateBlockChainWithSampleData()
	for i := range blocks {
		testutil.AssertEquals(t, testBlockchainWrapper.getBlock(uint64(i)), blocks[i])
	}
}

func TestIndexes_GetBlockByBlockHash(t *testing.T) {
	testDBWrapper.CreateFreshDB(t)
	testBlockchainWrapper := newTestBlockchainWrapper(t)
	blocks, _ := testBlockchainWrapper.populateBlockChainWithSampleData()
	for i := range blocks {
		blockHash, _ := blocks[i].GetHash()
		testutil.AssertEquals(t, testBlockchainWrapper.getBlockByHash(blockHash), blocks[i])
	}
}

func TestIndexes_GetTransactionByBlockNumberAndTxIndex(t *testing.T) {
	testDBWrapper.CreateFreshDB(t)
	testBlockchainWrapper := newTestBlockchainWrapper(t)
	blocks, _ := testBlockchainWrapper.populateBlockChainWithSampleData()
	for i, block := range blocks {
		for j, tx := range block.GetTransactions() {
			testutil.AssertEquals(t, testBlockchainWrapper.getTransaction(uint64(i), uint64(j)), tx)
		}
	}
}

func TestIndexes_GetTransactionByBlockHashAndTxIndex(t *testing.T) {
	testDBWrapper.CreateFreshDB(t)
	testBlockchainWrapper := newTestBlockchainWrapper(t)
	blocks, _ := testBlockchainWrapper.populateBlockChainWithSampleData()
	for _, block := range blocks {
		blockHash, _ := block.GetHash()
		for j, tx := range block.GetTransactions() {
			testutil.AssertEquals(t, testBlockchainWrapper.getTransactionByBlockHash(blockHash, uint64(j)), tx)
		}
	}
}

func TestIndexes_GetTransactionByUUID(t *testing.T) {
	testDBWrapper.CreateFreshDB(t)
	testBlockchainWrapper := newTestBlockchainWrapper(t)
	tx1, uuid1 := buildTestTx()
	tx2, uuid2 := buildTestTx()
	block1 := protos.NewBlock([]*protos.Transaction{tx1, tx2})
	testBlockchainWrapper.addNewBlock(block1, []byte("stateHash1"))

	tx3, uuid3 := buildTestTx()
	tx4, uuid4 := buildTestTx()
	block2 := protos.NewBlock([]*protos.Transaction{tx3, tx4})
	testBlockchainWrapper.addNewBlock(block2, []byte("stateHash2"))

	testutil.AssertEquals(t, testBlockchainWrapper.getTransactionByUUID(uuid1), tx1)
	testutil.AssertEquals(t, testBlockchainWrapper.getTransactionByUUID(uuid2), tx2)
	testutil.AssertEquals(t, testBlockchainWrapper.getTransactionByUUID(uuid3), tx3)
	testutil.AssertEquals(t, testBlockchainWrapper.getTransactionByUUID(uuid4), tx4)
}
