package launcher

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	util "codeberg.org/telecter/cmd-launcher/internal"
	"codeberg.org/telecter/cmd-launcher/internal/api"
	"codeberg.org/telecter/cmd-launcher/internal/auth"
)

type LaunchOptions struct {
	ModLoader       string
	QuickPlayServer string
}

func GetVersionDir(rootDir string, version string) string {
	path := filepath.Join(rootDir, "versions", version)
	return path
}

func run(args []string) error {
	cmd := exec.Command("java", args...)
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	cmd.Start()

	go func() {
		io.Copy(os.Stdout, stdout)
	}()
	go func() {
		io.Copy(os.Stderr, stderr)
	}()

	return cmd.Wait()
}

func Launch(version string, rootDir string, options LaunchOptions, authData auth.MinecraftLoginData) error {
	var meta api.VersionMeta
	// TODO: fix repeating code here
	if version == "" {
		var err error
		meta, err = api.GetVersionMeta("")
		if err != nil {
			return err
		}
		version = meta.ID
	}
	versionDir := GetVersionDir(rootDir, version)
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		return fmt.Errorf("failed to create game directory: %s", err)
	}
	if data, err := os.ReadFile(filepath.Join(versionDir, version+".json")); err == nil {
		json.Unmarshal(data, &meta)
	} else {
		meta, err = api.GetVersionMeta(version)
		if err != nil {
			return err
		}
		os.WriteFile(filepath.Join(versionDir, version+".json"), data, 0644)
	}

	libraries, err := getLibraries(meta.Libraries, rootDir)
	if err != nil {
		return fmt.Errorf("error downloading libraries: %s", err)
	}

	var loaderMeta api.FabricMeta
	if options.ModLoader != "" {
		var url string
		switch options.ModLoader {
		case "fabric":
			url = api.FabricURLPrefix
		case "quilt":
			url = api.QuiltURLPrefix
		default:
			return fmt.Errorf("invalid mod loader")
		}
		if data, err := os.ReadFile(filepath.Join(versionDir, options.ModLoader+".json")); err == nil {
			json.Unmarshal(data, &loaderMeta)
		} else {
			loaderMeta, err = api.GetLoaderMeta(url, version)
			if err != nil {
				return err
			}
			data, _ := json.Marshal(loaderMeta)
			os.WriteFile(filepath.Join(versionDir, options.ModLoader+".json"), data, 0644)
		}
		loaderLibraries, err := getLibraries(loaderMeta.Libraries, rootDir)
		if err != nil {
			return fmt.Errorf("error downloading loader libraries: %s", err)
		}
		libraries = append(libraries, loaderLibraries...)
	}

	if err = getAssets(meta, rootDir); err != nil {
		return fmt.Errorf("error downloading assets: %s", err)
	}

	if err := util.DownloadFile(meta.Downloads.Client.URL, filepath.Join(versionDir, version+".jar")); err != nil {
		return fmt.Errorf("error downloading client: %s", err)
	}
	libraries = append(libraries, filepath.Join(versionDir, version+".jar"))

	jvmArgs := []string{"-cp", strings.Join(libraries, ":")}

	if runtime.GOOS == "darwin" {
		jvmArgs = append(jvmArgs, "-XstartOnFirstThread")
	}
	if options.ModLoader != "" {
		jvmArgs = append(jvmArgs, loaderMeta.Arguments.Jvm...)
		jvmArgs = append(jvmArgs, loaderMeta.MainClass)
	} else {
		jvmArgs = append(jvmArgs, meta.MainClass)
	}

	gameArgs := []string{"--username", authData.Username, "--accessToken", authData.Token, "--gameDir", versionDir, "--assetsDir", filepath.Join(rootDir, "assets"), "--assetIndex", meta.AssetIndex.ID, "--version", version, "--versionType", meta.Type}
	if authData.UUID != "" {
		gameArgs = append(gameArgs, "--uuid", authData.UUID)
	}
	os.Chdir(versionDir)
	return run(append(jvmArgs, gameArgs...))
}
