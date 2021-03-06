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

package statemgmt

import (
	"bytes"

	"github.com/op/go-logging"
)

var logger = logging.MustGetLogger("statemgmt")

var stateKeyDelimiter = []byte{0x00}

// ConstructCompositeKey returns a []byte that uniquely represents a given chaincodeID and key.
// This assumes that chaincodeID and key does not contain a nil byte
func ConstructCompositeKey(chaincodeID string, key string) []byte {
	compositeKey := []byte(chaincodeID)
	compositeKey = append(compositeKey, stateKeyDelimiter...)
	compositeKey = append(compositeKey, []byte(key)...)
	return compositeKey
}

// DecodeCompositeKey decodes the compositeKey constructed by ConstructCompositeKey method
// back to the original chaincodeID and key form
func DecodeCompositeKey(compositeKey []byte) (string, string) {
	split := bytes.Split(compositeKey, stateKeyDelimiter)
	return string(split[0]), string(split[1])
}

// Copy returns a copy of given bytes
func Copy(src []byte) []byte {
	dest := make([]byte, len(src))
	copy(dest, src)
	return dest
}
