package learned

import (
	"slices"
	"testing"
)

func frozenCounter(t *testing.T) *Executor {
	t.Helper()
	key, err := PublicKey("../../experiments/counter/run-0001/trusted-key.hex")
	if err != nil {
		t.Fatal(err)
	}
	e, err := Load("../../experiments/counter/run-0001/artifact.json", key)
	if err != nil {
		t.Fatal(err)
	}
	return e
}
func TestCompositionCertificateAndReducedWarmWork(t *testing.T) {
	e := frozenCounter(t)
	concrete, _, err := e.NewComposer("learned", "concrete")
	if err != nil {
		t.Fatal(err)
	}
	certified, proof, err := e.NewComposer("learned", "certified")
	if err != nil {
		t.Fatal(err)
	}
	if !proof.Valid || proof.Cases != 4386 || proof.Relation.IntegerPrimitives != 17028 || proof.BuildNS <= 0 {
		t.Fatal("incomplete certificate", proof)
	}
	s := Sequence{Initial: 12, Steps: []Operation{{ID: "Counter.increment", Delta: 3}, {ID: "Counter.read"}, {ID: "Counter.increment", Delta: -7}, {ID: "Counter.increment", Delta: 0}, {ID: "Counter.read"}}}
	a, b := concrete.Run(s), certified.Run(s)
	if !a.Accepted || !b.Accepted || a.State != 8 || b.State != 8 || a.Version != 2 || b.Version != 2 || !slices.Equal(a.Results, []int64{15, 15, 8, 8, 8}) || !slices.Equal(a.Results, b.Results) || !slices.Equal(b.Writes, []int64{15, 8, 8}) {
		t.Fatal("composition mismatch", a, b)
	}
	if a.Work.Relation.IntegerPrimitives != 12 || a.Timing.CheckerCalls != 5 || b.Work.Relation.IntegerPrimitives != 0 || b.Timing.CheckerCalls != 0 || b.Work.CertificateUses != 5 || b.Work.DomainChecks == 0 || b.Work.PredictionChecks != 15 {
		t.Fatal("boundary work accounting wrong", a, b)
	}
	// Mutation of a separately owned executor cannot alter the sealed plan's model
	// or relation metadata. The proof must never attach to a different pair.
	e.artifact.Model.Weights[0][0] = 1
	e.artifact.Contracts.Methods[0].Return.Op = "null"
	if !certified.Run(s).Accepted || !concrete.Run(s).Accepted {
		t.Fatal("plan failed to seal its proof inputs")
	}
}
func TestCompositionRejectionAndCertificateBinding(t *testing.T) {
	e := frozenCounter(t)
	for _, engine := range []string{"zero", "shuffled"} {
		p, proof, err := e.NewComposer(engine, "certified")
		if err == nil || p != nil || proof.Valid || proof.FailureCase == "" {
			t.Fatal("invalid model certified", engine, proof)
		}
	}
	learned, proof, err := e.NewComposer("learned", "certified")
	if err != nil {
		t.Fatal(err)
	}
	_, lookupProof, err := e.NewComposer("lookup", "certified")
	if err != nil || lookupProof.Key == proof.Key {
		t.Fatal("certificate reused across engines", err)
	}
	for _, s := range []Sequence{
		{Initial: 64, Steps: []Operation{{ID: "Counter.increment", Delta: 16}, {ID: "Counter.increment", Delta: -16}}},
		{Initial: 0, Steps: []Operation{{ID: "Counter.read", Delta: 1}}},
		{Initial: 0, Steps: []Operation{{ID: "Other.run"}}},
		{Initial: 0, Steps: make([]Operation, 17)},
		{Initial: 0},
	} {
		out := learned.Run(s)
		if out.Accepted || out.State != s.Initial || out.Version != 1 || len(out.Results) != 0 || len(out.Writes) != 0 {
			t.Fatal("rejected sequence leaked state", out)
		}
	}
	// A final output may leave the input domain; only a subsequent invocation
	// would require certification at that state. Last-step writes remain exact.
	out := learned.Run(Sequence{Initial: 64, Steps: []Operation{{ID: "Counter.increment", Delta: 16}}})
	if !out.Accepted || out.State != 80 {
		t.Fatal("last result unnecessarily rejected", out)
	}
	if out := new(Composer).Run(Sequence{Initial: 0, Steps: []Operation{{ID: "Counter.read"}}}); out.Accepted {
		t.Fatal("uninitialized composer bypassed gate")
	}
}
