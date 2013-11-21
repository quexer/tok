/**
 *
 */
package ri

import "testing"

func TestPing(t *testing.T) {
	a := make([]byte, 0)
	if "" != string(a) {
		t.Error("zero byte doesn't equal empty string")
	}
}
