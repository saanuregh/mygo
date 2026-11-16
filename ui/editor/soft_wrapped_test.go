package editor

import "testing"

// SoftWrapped is SoftWrap, flipped by each Toggle Soft Wrap.
func TestSoftWrappedCountsTheToggle(t *testing.T) {
	ct := newEditorTest(t, "", "a|")
	for _, wrap := range []bool{false, true} {
		ct.st.SoftWrap = wrap
		if ct.st.SoftWrapped() != wrap {
			t.Fatalf("SoftWrap %v: SoftWrapped %v", wrap, !wrap)
		}
		ct.run(ToggleSoftWrap)
		if ct.st.SoftWrapped() == wrap {
			t.Fatalf("SoftWrap %v toggled: SoftWrapped still %v", wrap, wrap)
		}
		ct.run(ToggleSoftWrap)
	}
}
