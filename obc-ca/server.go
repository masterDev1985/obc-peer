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

package main

import (
	"io/ioutil"
	"os"
	"sync"

	"github.com/openblockchain/obc-peer/obc-ca/obcca"
	"github.com/spf13/viper"
)

func main() {
	viper.AutomaticEnv()
	viper.SetConfigName("obcca")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./")
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}

	obcca.LogInit(ioutil.Discard, os.Stdout, os.Stdout, os.Stderr, os.Stdout)

	eca := obcca.NewECA()
	defer eca.Close()

	tca := obcca.NewTCA(eca)
	defer tca.Close()

	tlsca := obcca.NewTLSCA()
	defer tlsca.Close()

	var wg sync.WaitGroup
	eca.Start(&wg)
	tca.Start(&wg)
	tlsca.Start(&wg)

	wg.Wait()
}
