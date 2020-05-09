package fub

import "testing"

func TestEncode(t *testing.T) {
	m := InitResponse{
		Name: "Foo",
		Port: 4711,
	}
	bs, err := EncodeMessage(m)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("encoded: (%s)", string(bs))

	mc, err := DecodeMessage(bs)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%#v", mc)

	mack := InitResponse{}
	bs, err = EncodeMessage(mack)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("encoded: (%s)", string(bs))
}
