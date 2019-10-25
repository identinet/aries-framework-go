/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bdd

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/godog"

	"github.com/hyperledger/aries-framework-go/pkg/common/log"
	"github.com/hyperledger/aries-framework-go/test/bdd/dockerutil"
)

var composition []*dockerutil.Composition
var composeFiles = []string{"./fixtures/sidetree-node", "./fixtures/agent"}

func TestMain(m *testing.M) {
	// default is to run all tests with tag @all
	tags := "all"
	flag.Parse()

	format := "progress"
	if getCmdArg("test.v") == "true" {
		format = "pretty"
	}

	runArg := getCmdArg("test.run")
	if runArg != "" {
		tags = runArg
	}

	agentLogLevel := os.Getenv("AGENT_LOG_LEVEL")
	if agentLogLevel != "" {
		logLevel, err := log.ParseLevel(agentLogLevel)
		if err != nil {
			panic(err)
		}
		log.SetLevel(os.Getenv("AGENT_LOG_MODULE"), logLevel)
	}

	initBDDConfig()

	status := godog.RunWithOptions("godogs", func(s *godog.Suite) {
		s.BeforeSuite(func() {

			if os.Getenv("DISABLE_COMPOSITION") != "true" {

				// Need a unique name, but docker does not allow '-' in names
				composeProjectName := strings.Replace(generateUUID(), "-", "", -1)

				for _, v := range composeFiles {
					newComposition, err := dockerutil.NewComposition(composeProjectName, "docker-compose.yml", v)
					if err != nil {
						panic(fmt.Sprintf("Error composing system in BDD context: %s", err))
					}
					composition = append(composition, newComposition)
				}
				fmt.Println("docker-compose up ... waiting for containers to start ...")
				testSleep := 5
				if os.Getenv("TEST_SLEEP") != "" {
					testSleep, _ = strconv.Atoi(os.Getenv("TEST_SLEEP"))
				}
				fmt.Printf("*** testSleep=%d", testSleep)
				time.Sleep(time.Second * time.Duration(testSleep))
			}

		})
		s.AfterSuite(func() {
			for _, c := range composition {
				if c != nil {
					if err := c.GenerateLogs(c.Dir, c.ProjectName+".log"); err != nil {
						panic(err)
					}
					if _, err := c.Decompose(c.Dir); err != nil {
						panic(err)
					}
				}
			}
		})
		FeatureContext(s)
	}, godog.Options{
		Tags:          tags,
		Format:        format,
		Paths:         []string{"features"},
		Randomize:     time.Now().UTC().UnixNano(), // randomize scenario execution order
		Strict:        true,
		StopOnFailure: true,
	})

	if st := m.Run(); st > status {
		status = st
	}
	os.Exit(status)
}

func getCmdArg(argName string) string {
	cmdTags := flag.CommandLine.Lookup(argName)
	if cmdTags != nil && cmdTags.Value != nil && cmdTags.Value.String() != "" {
		return cmdTags.Value.String()
	}
	return ""
}

func FeatureContext(s *godog.Suite) {
	context, err := NewContext()
	if err != nil {
		panic(fmt.Sprintf("Error returned from NewBDDContext: %s", err))
	}

	// set dynamic args
	context.Args[SideTreeURL] = "http://localhost:48326/.sidetree/document"
	context.Args[DIDDocPath] = "fixtures/sidetree-node/config/didDocument.json"

	// TODO below 2 env variables to be removed as part of issue #572
	context.Args[AliceAgentHost] = "alice.agent.example.com"
	context.Args[BobAgentHost] = "bob.agent.example.com"

	// Context is shared between tests
	NewAgentSDKSteps(context).RegisterSteps(s)
	NewAgentControllerSteps(context).RegisterSteps(s)
	NewDIDExchangeSteps(context).RegisterSteps(s)
	NewDIDResolverSteps(context).RegisterSteps(s)

}

func initBDDConfig() {
}
