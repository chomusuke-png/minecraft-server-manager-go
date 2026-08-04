package java

import "testing"

func TestRequire(t *testing.T) {
	cases := []struct {
		mcVersion string
		want      Requirement
	}{
		// La era pre-1.17 tiene tope: no corre en Java 17+.
		{"1.8.9", Requirement{Min: 8, Max: 11}},
		{"1.12.2", Requirement{Min: 8, Max: 11}},
		{"1.16.5", Requirement{Min: 8, Max: 11}},
		{"1.17", Requirement{Min: 17}},
		{"1.17.1", Requirement{Min: 17}},
		{"1.19.1", Requirement{Min: 17}},
		{"1.20", Requirement{Min: 17}},
		{"1.20.4", Requirement{Min: 17}},
		{"1.20.5", Requirement{Min: 21}}, // el corte de Java 21 cae en un patch, no en un minor
		{"1.20.6", Requirement{Min: 21}},
		{"1.21", Requirement{Min: 21}},
		{"1.21.4", Requirement{Min: 21}},
		{"1.20.5-pre1", Requirement{Min: 21}}, // el sufijo no debe romper el parseo del patch
		{"", Requirement{}},
		{"snapshot", Requirement{}},
		{"23w31a", Requirement{}},
	}

	for _, c := range cases {
		if got := Require(c.mcVersion); got != c.want {
			t.Errorf("Require(%q) = %+v, want %+v", c.mcVersion, got, c.want)
		}
	}
}

// Verificado a mano: Forge 1.19.1 arranca en Java 21, así que el requisito de la
// era moderna es un mínimo y no un valor exacto.
func TestModernRequirementHasNoUpperBound(t *testing.T) {
	modern := Require("1.19.1")
	for _, major := range []int{17, 21, 25} {
		if !modern.Satisfies(major) {
			t.Errorf("1.19.1 debería aceptar Java %d", major)
		}
	}
	if modern.Satisfies(8) {
		t.Error("1.19.1 no debería aceptar Java 8")
	}
}

// La era pre-1.17 sí tiene tope superior.
func TestLegacyRequirementRejectsModernJava(t *testing.T) {
	legacy := Require("1.16.5")
	for _, major := range []int{8, 11} {
		if !legacy.Satisfies(major) {
			t.Errorf("1.16.5 debería aceptar Java %d", major)
		}
	}
	for _, major := range []int{17, 21} {
		if legacy.Satisfies(major) {
			t.Errorf("1.16.5 no debería aceptar Java %d", major)
		}
	}
}

func TestRequirementString(t *testing.T) {
	cases := []struct {
		req  Requirement
		want string
	}{
		{Requirement{}, "cualquier Java"},
		{Requirement{Min: 17}, "Java 17 o superior"},
		{Requirement{Min: 8, Max: 11}, "Java 8 a 11"},
		{Requirement{Min: 8, Max: 8}, "Java 8"},
	}
	for _, c := range cases {
		if got := c.req.String(); got != c.want {
			t.Errorf("%+v.String() = %q, want %q", c.req, got, c.want)
		}
	}
}

// Una versión no reconocida no debe bloquear el arranque.
func TestUnknownRequirementSatisfiedByAnything(t *testing.T) {
	unknown := Require("23w31a")
	if unknown.Min != 0 {
		t.Fatalf("Min = %d, want 0", unknown.Min)
	}
	for _, major := range []int{8, 17, 21} {
		if !unknown.Satisfies(major) {
			t.Errorf("un requisito desconocido debería aceptar Java %d", major)
		}
	}
}

// DetectMajor tiene que entender el formato legacy 1.8.0_x y el moderno.
func TestDetectMajorParsesBothFormats(t *testing.T) {
	cases := []struct {
		output string
		want   int
	}{
		{`openjdk version "17.0.20" 2026-01-20`, 17},
		{`openjdk version "1.8.0_502"`, 8},
		{`java version "21.0.12" 2025-07-15 LTS`, 21},
	}

	for _, c := range cases {
		match := reportedVersionPattern.FindStringSubmatch(c.output)
		if match == nil {
			t.Errorf("no matcheó %q", c.output)
			continue
		}
		got := 0
		if match[1] == "1" {
			got = atoiOrZero(match[2])
		} else {
			got = atoiOrZero(match[1])
		}
		if got != c.want {
			t.Errorf("de %q se sacó %d, want %d", c.output, got, c.want)
		}
	}
}

func atoiOrZero(value string) int {
	parsed, _ := leadingInt(value)
	return parsed
}
