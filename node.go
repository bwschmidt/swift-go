/* ****************************************************************************
 * Copyright 2020 51 Degrees Mobile Experts Limited (51degrees.com)
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
	"bytes"
	"encoding/base64"
	"fmt"
	"hash/fnv"
	"net/http"
	"sort"
	"time"
)

const (
	roleAccess  = iota // The node responds to server initiated access requests
	roleStorage = iota // The node can be used for storage operations
)

type node struct {
	network   string    // The name of the network the node belongs to
	domain    string    // The domain name associated with the node
	hash      uint32    // Number used to relate client IPs to node
	created   time.Time // The time that the node first came online
	expires   time.Time // The time that the node will retire from the network
	role      int       // The role the node has in the network
	secrets   []*secret // All the secrets associated with the node
	scrambler *secret   // Secret used to scramble data with fixed nonce
	nonce     []byte    // Fixed nonce used with the scrambler
	alive     bool      // True if the node is reachable via a HTTP request
}

func (n *node) Domain() string { return n.domain }

func newNode(
	network string,
	domain string,
	created time.Time,
	expires time.Time,
	role int,
	scrambleKey string) (*node, error) {
	h := fnv.New32a()
	h.Write([]byte(domain))
	s, err := newSecretFromKey(scrambleKey, created)
	if err != nil {
		return nil, err
	}
	n := node{
		network,
		domain,
		h.Sum32(),
		created,
		expires,
		role,
		make([]*secret, 0),
		s,
		makeNonce(s, []byte(domain)),
		false}
	return &n, nil
}

func makeNonce(s *secret, d []byte) []byte {
	n := make([]byte, s.crypto.gcm.NonceSize())
	c := 0
	for i := 0; i < len(n); i++ {
		n[i] = d[c]
		c++
		if c >= len(d) {
			c = 0
		}
	}
	return n
}

func (n *node) isActive() bool {
	return n.expires.After(time.Now().UTC()) && len(n.secrets) > 0
}

func (n *node) unscramble(s string) (string, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	d, err := n.scrambler.crypto.decrypt(b)
	if err != nil {
		return "", err
	}
	return string(d), err
}

func (n *node) scramble(s string) string {
	return base64.RawURLEncoding.EncodeToString(
		n.scrambler.crypto.encryptWithNonce([]byte(s), n.nonce))
}

func (n *node) encrypt(d []byte) ([]byte, error) {
	s, err := n.getSecret()
	if err != nil {
		return nil, err
	}
	return s.crypto.compressAndEncrypt(d)
}

func (n *node) decrypt(d []byte) ([]byte, error) {
	var err error
	for _, s := range n.secrets {
		b, err := s.crypto.decryptAndDecompress(d)
		if err == nil {
			return b, nil
		}
	}
	return nil, err
}

func (n *node) getValueFromCookie(c *http.Cookie) (*pair, error) {
	var p pair
	v, err := base64.RawURLEncoding.DecodeString(c.Value)
	if err != nil {
		return nil, err
	}
	d, err := n.decrypt(v)
	if err != nil {
		return nil, err
	}
	if len(d) == 0 {
		return nil, fmt.Errorf("Value for cookie '%s' zero length", c.Name)
	}
	b := bytes.NewBuffer(d)
	p.cookieWriteTime, err = readTime(b)
	if err != nil {
		return nil, fmt.Errorf("Time for cookie '%s' invalid", c.Name)
	}
	err = p.setFromBuffer(b)
	if err != nil {
		return nil, fmt.Errorf(
			"Value for cookie '%s' error '%s'",
			c.Name,
			err.Error())
	}
	return &p, nil
}

func (n *node) addSecret(secret *secret) {
	n.secrets = append(n.secrets, secret)
}

func (n *node) getSecret() (*secret, error) {
	if n == nil {
		fmt.Println("Null node")
	}
	if len(n.secrets) > 0 {
		return n.secrets[0], nil
	}
	return nil, fmt.Errorf("No secrets for node '%s'", n.domain)
}

func (n *node) sortSecrets() {
	sort.Slice(n.secrets, func(i, j int) bool {
		return n.secrets[i].timeStamp.Sub(n.secrets[j].timeStamp) < 0
	})
}
