/* ****************************************************************************
 * Copyright 2020 51 Degrees Mobile Experts Limited
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not
 * use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
 * WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
 * License for the specific language governing permissions and limitations
 * under the License.
 * ***************************************************************************/

package swift

import (
	"fmt"
	"time"
)

func newVolatileTest() (volatile, error) {
	var v volatile
	v.init()
	for i := 1; i <= 10; i++ {
		_, err := v.testAddStorage(i)
		if err != nil {
			return v, err
		}
	}
	return v, nil
}

func (v *volatile) testAddStorage(index int) (*node, error) {
	s, err := newSecret()
	if err != nil {
		return nil, err
	}
	n := node{
		"network",
		fmt.Sprintf("test-%d.com", index),
		0,
		time.Now(),
		time.Now(),
		0,
		make([]*secret, 1),
		s,
		make([]byte, s.crypto.gcm.NonceSize()),
		true}
	x, err := newSecret()
	if err != nil {
		return nil, err
	}
	n.secrets = append(n.secrets, x)
	v.setNode(&n)
	return &n, nil
}