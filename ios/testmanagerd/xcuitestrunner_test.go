//go:build !fast
// +build !fast

package testmanagerd_test

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/danielpaulus/go-ios/ios"
	"github.com/danielpaulus/go-ios/ios/imagemounter"
	"github.com/danielpaulus/go-ios/ios/installationproxy"
	"github.com/danielpaulus/go-ios/ios/testmanagerd"
	"github.com/danielpaulus/go-ios/ios/zipconduit"
	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

const (
	wdapath       = "../../testdata/wda.ipa"
	wdaSignedPath = "../../testdata/wda-signed.ipa"
)

const signerPath = "../../testdata/app-signer-mac"

const (
	wdaSuccessLogMessage = "ServerURLHere"
	bundleId             = "com.facebook.WebDriverAgentRunner.xctrunner"
)

func TestXcuiTest(t *testing.T) {
	hook := test.NewGlobal()
	log.SetLevel(log.DebugLevel)

	device, err := ios.GetDevice("")
	if err != nil {
		t.Fatal(err)
	}
	// mounts&downloads developer image if needed
	err = imagemounter.FixDevImage(device, ".")
	if err != nil {
		t.Error(err)
		return
	}

	// get wda to the device if not installed already
	err = signAndInstall(device)
	if err != nil {
		t.Error(err)
		return
	}

	bundleID, testbundleID, xctestconfig := "com.facebook.WebDriverAgentRunner.xctrunner", "com.facebook.WebDriverAgentRunner.xctrunner", "WebDriverAgentRunner.xctest"
	var wdaargs []string
	var wdaenv []string
	go func() {
		err := testmanagerd.RunXCUIWithBundleIdsCtx(context.Background(), bundleID, testbundleID, xctestconfig, device, wdaargs, wdaenv)
		if err != nil {
			log.WithFields(log.Fields{"error": err}).Fatal("Failed running WDA")
		}
	}()

	wdaStarted := make(chan bool)
	pollLogs(hook, wdaStarted)

	select {
	case <-time.After(time.Second * 50):
		t.Error("timeout")
		return
	case <-wdaStarted:
		log.Info("wda started successfully")
	}

	log.Infof("done")

	err = testmanagerd.CloseXCUITestRunner()
	if err != nil {
		log.Error("Failed closing wda-testrunner")
		t.Fail()
	}
}

func pollLogs(hook *test.Hook, wdaStarted chan bool) {
	go func() {
		for {
			entries := hook.AllEntries()
			for _, e := range entries {
				if strings.Contains(e.Message, wdaSuccessLogMessage) {
					wdaStarted <- true
					return
				}
			}
		}
	}()
}

func signAndInstall(device ios.DeviceEntry) error {
	svc, _ := installationproxy.New(device)
	response, err := svc.BrowseUserApps()
	for _, info := range response {
		if bundleId == info.CFBundleIdentifier {
			log.Info("wda installed, skipping installation")
			return nil
		}
	}

	err = SignWda(device)
	if err != nil {
		return err
	}
	conn, err := zipconduit.New(device)
	if err != nil {
		return err
	}
	defer conn.Close()
	return conn.SendFile(wdaSignedPath)
}

func SignWda(device ios.DeviceEntry) error {
	cmd := exec.Command(signerPath,
		fmt.Sprintf("--udid=%s", device.Properties.SerialNumber),
		"--p12password=a",
		"--profilespath=../../testdata",
		fmt.Sprintf("--ipa=%s", wdapath),
		fmt.Sprintf("--output=%s", wdaSignedPath),
	)
	_, err := cmd.CombinedOutput()
	return err
}
