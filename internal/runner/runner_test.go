package runner

import (
	"bytes"
	"slices"
	"testing"

	"minecraft-manager/internal/config"
	"minecraft-manager/internal/instance"
)

func testRunner() *Runner {
	return New(&config.Config{JavaPath: "java", JarName: "server.jar", RAMGB: 4})
}

// Sin launch_args el arranque sigue siendo el clásico de -jar.
func TestBuildJavaArgsJarLoader(t *testing.T) {
	got := testRunner().buildJavaArgs(&instance.InstanceMeta{LoaderType: "paper"}, 6)

	want := []string{"-Xmx6G", "-Xms6G", "-jar", "server.jar", "nogui"}
	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

// La RAM va después de @user_jvm_args.txt (para ganarle a un -Xmx del usuario) y
// antes del args file del loader (que declara la main class).
func TestBuildJavaArgsForgeRAMPosition(t *testing.T) {
	meta := &instance.InstanceMeta{
		LoaderType: "forge",
		LaunchArgs: []string{
			"@user_jvm_args.txt",
			"@libraries/net/minecraftforge/forge/1.20.1-47.2.0/win_args.txt",
			"nogui",
		},
	}

	got := testRunner().buildJavaArgs(meta, 8)

	want := []string{
		"@user_jvm_args.txt",
		"-Xmx8G",
		"-Xms8G",
		"@libraries/net/minecraftforge/forge/1.20.1-47.2.0/win_args.txt",
		"nogui",
	}
	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBuildJavaArgsForgeWithoutUserJvmArgs(t *testing.T) {
	meta := &instance.InstanceMeta{
		LoaderType: "forge",
		LaunchArgs: []string{"@libraries/forge/win_args.txt", "nogui"},
	}

	got := testRunner().buildJavaArgs(meta, 4)

	want := []string{"-Xmx4G", "-Xms4G", "@libraries/forge/win_args.txt", "nogui"}
	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

// El sniffer tiene que reconocer las dos formas en que falla un Java equivocado, y
// dejar pasar la salida intacta.
func TestJavaMismatchSniffer(t *testing.T) {
	cases := []struct {
		name string
		line string
		want bool
	}{
		{
			"java viejo",
			"java.lang.UnsupportedClassVersionError: net/minecraft/server/Main has been compiled by a more recent version of the Java Runtime",
			true,
		},
		{
			// Verificado con Forge 1.16.5 sobre Java 17.
			"java nuevo para un loader pre-1.17",
			"Exception in thread \"main\" java.lang.IllegalAccessError: class cpw.mods.modlauncher.SecureJarHandler cannot access class sun.security.util.ManifestEntryVerifier (in module java.base) because module java.base does not export sun.security.util to unnamed module",
			true,
		},
		{
			"arranque normal",
			"[Server thread/INFO]: Done (53.281s)! For help, type \"help\"",
			false,
		},
		{
			"crash no relacionado con Java",
			"java.lang.OutOfMemoryError: Java heap space",
			false,
		},
	}

	for _, c := range cases {
		var passedThrough bytes.Buffer
		sniffer := &javaMismatchSniffer{out: &passedThrough}

		if _, err := sniffer.Write([]byte(c.line)); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if sniffer.detected != c.want {
			t.Errorf("%s: detected = %v, want %v", c.name, sniffer.detected, c.want)
		}
		if passedThrough.String() != c.line {
			t.Errorf("%s: la salida no pasó intacta", c.name)
		}
	}
}

// buildJavaArgs no debe mutar el meta que después se persiste en instance.json.
func TestBuildJavaArgsDoesNotMutateMeta(t *testing.T) {
	original := []string{"@user_jvm_args.txt", "@libraries/forge/win_args.txt", "nogui"}
	meta := &instance.InstanceMeta{LoaderType: "forge", LaunchArgs: slices.Clone(original)}

	testRunner().buildJavaArgs(meta, 4)

	if !slices.Equal(meta.LaunchArgs, original) {
		t.Errorf("launch_args mutado: got %q, want %q", meta.LaunchArgs, original)
	}
}
