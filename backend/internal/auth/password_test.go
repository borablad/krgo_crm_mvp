package auth

import "testing"

func TestPasswordRoundTrip(t *testing.T) {
	hash, err := HashPassword("секретный пароль 123")
	if err != nil {
		t.Fatal(err)
	}
	ok, err := VerifyPassword("секретный пароль 123", hash)
	if err != nil || !ok {
		t.Fatalf("верный пароль не принят: ok=%v err=%v", ok, err)
	}
	ok, err = VerifyPassword("другой пароль", hash)
	if err != nil || ok {
		t.Fatalf("неверный пароль принят: ok=%v err=%v", ok, err)
	}
}

func TestPasswordHashesDiffer(t *testing.T) {
	a, _ := HashPassword("one")
	b, _ := HashPassword("one")
	if a == b {
		t.Fatal("одинаковые хэши для одного пароля: соль не случайна")
	}
}

func TestVerifyPasswordBadFormat(t *testing.T) {
	if _, err := VerifyPassword("x", "not-a-hash"); err == nil {
		t.Fatal("ожидалась ошибка формата хэша")
	}
}

func TestRolePermissions(t *testing.T) {
	cases := []struct {
		role Role
		perm Permission
		want bool
	}{
		{RoleAdmin, PermUsersManage, true},
		{RoleManager, PermTreasuryApprove, true},
		{RoleEmployee, PermTreasuryApprove, false},
		{RoleAccountant, PermCRMRead, false},
		{RoleStorekeeper, PermWMSWrite, true},
		{RoleStorekeeper, PermCRMRead, false},
		{Role("unknown"), PermDirectoriesRead, false},
	}
	for _, c := range cases {
		if got := c.role.Can(c.perm); got != c.want {
			t.Errorf("%s.Can(%s) = %v, ожидалось %v", c.role, c.perm, got, c.want)
		}
	}
}
