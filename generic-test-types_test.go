package ecs_test

// genComp1..genComp16 are 16 distinct component types, used both by the
// generated smoke test (ecs/zz_gen_generic-api_test.go) and by the
// hand-written per-arity tests (new-entity_test.go, view_test.go): every
// generated family needs N distinct types to exercise arity N for real,
// and these are declared once here instead of once per generated test.
type (
	genComp1  struct{ V int }
	genComp2  struct{ V int }
	genComp3  struct{ V int }
	genComp4  struct{ V int }
	genComp5  struct{ V int }
	genComp6  struct{ V int }
	genComp7  struct{ V int }
	genComp8  struct{ V int }
	genComp9  struct{ V int }
	genComp10 struct{ V int }
	genComp11 struct{ V int }
	genComp12 struct{ V int }
	genComp13 struct{ V int }
	genComp14 struct{ V int }
	genComp15 struct{ V int }
	genComp16 struct{ V int }

	// genExcluded is never a type parameter of a generated test, so every
	// arity can exclude it with ecs.Without.
	genExcluded struct{ V int }
)
