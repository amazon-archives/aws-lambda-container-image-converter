// +build linux

package keyctl

import (
	"crypto/rand"
	"strings"
	"testing"
)

func TestSessionKeyring(t *testing.T) {

	token := make([]byte, 20)
	rand.Read(token)

	testname := "testname"
	keyring, err := SessionKeyring()
	if err != nil {
		t.Fatal(err)
	}
	_, err = keyring.Add(testname, token)
	if err != nil {
		t.Fatal(err)
	}
	key, err := keyring.Search(testname)
	if err != nil {
		t.Fatal(err)
	}
	data, err := key.Get()
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(token) {
		t.Errorf("Expected data %v, but get %v", token, data)
	}
	err = key.Unlink()
	if err != nil {
		t.Fatal(err)
	}
}

func TestUserKeyring(t *testing.T) {
	token := make([]byte, 20)
	rand.Read(token)

	testname := "testuser"

	userKeyring, err := UserKeyring()
	if err != nil {
		t.Fatal(err)
	}

	userKey, err := userKeyring.Add(testname, token)
	if err != nil {
		t.Fatal(err, userKey)
	}

	searchRet, err := userKeyring.Search(testname)
	if err != nil {
		t.Fatal(err)
	}
	if searchRet.Name != testname {
		t.Errorf("Expected data %v, but get %v", testname, searchRet.Name)
	}

	err = userKey.Unlink()
	if err != nil {
		t.Fatal(err)
	}
}

func TestLink(t *testing.T) {
	token := make([]byte, 20)
	rand.Read(token)

	testname := "testlink"

	userKeyring, err := UserKeyring()
	if err != nil {
		t.Fatal(err)
	}

	sessionKeyring, err := SessionKeyring()
	if err != nil {
		t.Fatal(err)
	}

	key, err := sessionKeyring.Add(testname, token)
	if err != nil {
		t.Fatal(err)
	}

	_, err = userKeyring.Search(testname)
	if err == nil {
		t.Fatalf("Expected error, but got key %v", testname)
	}
	ExpectedError := "required key not available"
	if err.Error() != ExpectedError {
		t.Fatal(err)
	}

	err = Link(userKeyring, key)
	if err != nil {
		t.Fatal(err)
	}
	_, err = userKeyring.Search(testname)
	if err != nil {
		t.Fatal(err)
	}

	err = key.Unlink()
	if err != nil {
		t.Fatal(err)
	}

	err = Unlink(userKeyring, key)
	if err != nil {
		t.Fatal(err)
	}
}

func TestUnlink(t *testing.T) {
	token := make([]byte, 20)
	rand.Read(token)

	testname := "testunlink"
	keyring, err := SessionKeyring()
	if err != nil {
		t.Fatal(err)
	}
	key, err := keyring.Add(testname, token)
	if err != nil {
		t.Fatal(err)
	}

	err = Unlink(keyring, key)
	if err != nil {
		t.Fatal(err)
	}

	_, err = keyring.Search(testname)
	ExpectedError := "required key not available"
	if err.Error() != ExpectedError {
		t.Fatal(err)
	}
}

func TestReadKeyring(t *testing.T) {
	token := make([]byte, 20)
	rand.Read(token)

	testname := "testuser"

	userKeyring, err := UserKeyring()
	if err != nil {
		t.Fatal(err)
	}

	userKey, err := userKeyring.Add(testname, token)
	if err != nil {
		t.Fatal(err, userKey)
	}
	keys, err := ReadUserKeyring()
	if err != nil {
		t.Fatal(err)
	}
	expectedKeyLen := 1
	if len(keys) != 1 {
		t.Errorf("expected to read %d userkeyring, but get %d", expectedKeyLen, len(keys))
	}
	err = Unlink(userKeyring, userKey)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDescribe(t *testing.T) {
	token := make([]byte, 20)
	rand.Read(token)

	testname := "testuser"

	userKeyring, err := UserKeyring()
	if err != nil {
		t.Fatal(err)
	}

	userKey, err := userKeyring.Add(testname, token)
	if err != nil {
		t.Fatal(err, userKey)
	}
	keyAttr, err := userKey.Describe()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(keyAttr, testname) {
		t.Errorf("expect description contains %s, but get %s", testname, keyAttr)
	}
	err = Unlink(userKeyring, userKey)
	if err != nil {
		t.Fatal(err)
	}
}
