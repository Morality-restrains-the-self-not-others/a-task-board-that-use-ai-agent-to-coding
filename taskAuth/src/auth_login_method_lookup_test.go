package main

import "testing"

func TestFindLoginMethodByEmailIgnoresArchivedUser(t *testing.T) {
	setupAuthTestDB(t)
	uid, _, err := createUserWithEmailLogin("archived-occ@example.com", "Passw0rd!")
	if err != nil {
		t.Fatalf("create email user: %v", err)
	}
	archiveUser(t, uid)

	lm, err := findLoginMethodByEmail("archived-occ@example.com")
	if err != nil {
		t.Fatalf("findLoginMethodByEmail: %v", err)
	}
	if lm != nil {
		t.Fatalf("archived email must not occupy, got %+v", lm)
	}
}

func TestFindLoginMethodByPhoneIgnoresArchivedUser(t *testing.T) {
	setupAuthTestDB(t)
	const national = "18965128763"
	uid, err := createUserWithPhoneLogin("+86", national, "OldPassw0rd!", "")
	if err != nil {
		t.Fatalf("create phone user: %v", err)
	}
	archiveUser(t, uid)

	lm, err := findLoginMethodByPhone("+86", national)
	if err != nil {
		t.Fatalf("findLoginMethodByPhone: %v", err)
	}
	if lm != nil {
		t.Fatalf("archived phone must not occupy, got %+v", lm)
	}
}
