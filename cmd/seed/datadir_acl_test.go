package main

import "testing"

func TestCheckProtectedDACL(t *testing.T) {
	const ok = "O:BAG:SYD:PAI(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)"
	tests := []struct {
		name    string
		owner   string
		sddl    string
		wantErr bool
	}{
		{"administrators own, SYSTEM and administrators only", sidAdministrators, ok, false},
		{"SYSTEM owns", sidSystem, ok, false},
		{"numeric trustees", sidAdministrators, "D:P(A;OICI;FA;;;S-1-5-18)(A;OICI;FA;;;S-1-5-32-544)", false},
		{"owned by a user", "S-1-5-21-1-2-3-1001", ok, true},
		{"inherits from its parent", sidAdministrators, "D:AI(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)", true},
		{"users can read", sidAdministrators, "D:PAI(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;FRFX;;;BU)", true},
		{
			"a named user has an entry", sidAdministrators,
			"D:PAI(A;OICI;FA;;;SY)(A;OICI;FA;;;S-1-5-21-1-2-3-1001)", true,
		},
		{"a deny entry", sidAdministrators, "D:PAI(D;OICI;FA;;;BU)(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)", true},
		{"empty access list", sidAdministrators, "D:P", true},
		{"no access list", sidAdministrators, "O:BAG:SY", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := checkProtectedDACL(tt.owner, tt.sddl); (err != nil) != tt.wantErr {
				t.Errorf("checkProtectedDACL error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
