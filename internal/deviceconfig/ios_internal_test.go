package deviceconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMergeIOS(t *testing.T) {
	t.Parallel()
	const running = "Building configuration...\n!\nhostname edge\n!\n" +
		"interface Gi1/0/1\n description old\n shutdown\n!\n" +
		"interface Gi1/0/2\n no shutdown\n!\n" +
		"ip access-list extended MGMT\n permit tcp any any eq 22\n!\nend\n"
	tests := []struct {
		name      string
		candidate string
		want      string
	}{
		{
			name:      "a new section goes before end",
			candidate: "vlan 20\n name voice\n",
			want: "Building configuration...\nhostname edge\n" +
				"interface Gi1/0/1\n description old\n shutdown\n" +
				"interface Gi1/0/2\n no shutdown\n" +
				"ip access-list extended MGMT\n permit tcp any any eq 22\n" +
				"vlan 20\n name voice\nend\n",
		},
		{
			name:      "a singleton replaces its predecessor, and no undoes a line",
			candidate: "hostname core\ninterface Gi1/0/1\n description uplink\n no shutdown\n",
			want: "Building configuration...\nhostname core\n" +
				"interface Gi1/0/1\n description uplink\n" +
				"interface Gi1/0/2\n no shutdown\n" +
				"ip access-list extended MGMT\n permit tcp any any eq 22\nend\n",
		},
		{
			name:      "a line replaces its negation",
			candidate: "interface Gi1/0/2\n shutdown\n",
			want: "Building configuration...\nhostname edge\n" +
				"interface Gi1/0/1\n description old\n shutdown\n" +
				"interface Gi1/0/2\n shutdown\n" +
				"ip access-list extended MGMT\n permit tcp any any eq 22\nend\n",
		},
		{
			name:      "no removes a section and a line given with arguments",
			candidate: "no ip access-list extended MGMT\ninterface Gi1/0/1\n no description\n",
			want: "Building configuration...\nhostname edge\n" +
				"interface Gi1/0/1\n shutdown\n" +
				"interface Gi1/0/2\n no shutdown\nend\n",
		},
		{
			name:      "a candidate's own end and separators are not lines",
			candidate: "!\nhostname edge\n!\nend\n",
			want: "Building configuration...\nhostname edge\n" +
				"interface Gi1/0/1\n description old\n shutdown\n" +
				"interface Gi1/0/2\n no shutdown\n" +
				"ip access-list extended MGMT\n permit tcp any any eq 22\nend\n",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tree := parseIOS(running)
			before := renderIOS(tree)
			require.Equal(t, tc.want, renderIOS(mergeIOS(tree, parseIOS(tc.candidate))))
			require.Equal(t, before, renderIOS(tree), "the running tree is left as it was")
		})
	}
}

func TestIndentXML(t *testing.T) {
	t.Parallel()
	in := `<configuration xmlns:junos="http://xml.juniper.net/junos/"><system junos:changed="1">` +
		`<host-name>a &amp; b</host-name><services><ssh/></services></system></configuration>`
	want := "<configuration xmlns:junos=\"http://xml.juniper.net/junos/\">\n" +
		"  <system junos:changed=\"1\">\n" +
		"    <host-name>a &amp; b</host-name>\n" +
		"    <services>\n" +
		"      <ssh/>\n" +
		"    </services>\n" +
		"  </system>\n" +
		"</configuration>\n"
	require.Equal(t, want, indentXML(in))
	require.Equal(t, want, indentXML(want), "already indented XML renders the same")
	require.Equal(t, "not <xml", indentXML("not <xml"))
}
