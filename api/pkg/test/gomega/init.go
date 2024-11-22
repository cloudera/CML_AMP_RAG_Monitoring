package lgomega

import (
	"fmt"
	"github.com/onsi/gomega"
	omegatypes "github.com/onsi/gomega/types"
	"regexp"
	"runtime/debug"
	"strings"
)

func init() {
	gomega.RegisterFailHandler(buildTestingTGomegaFailHandler())
}

// Copied from https://github.com/fgrosse/gomega-matchers/blob/master/testing_t_support.go

func buildTestingTGomegaFailHandler() omegatypes.GomegaFailHandler {
	return func(message string, callerSkip ...int) {
		skip := 2 // initial runtime/debug.Stack frame + gomega-matchers.buildTestingTGomegaFailHandler frame
		if len(callerSkip) > 0 {
			skip += callerSkip[0]
		}
		stackTrace := pruneStack(string(debug.Stack()), skip)
		stackTrace = strings.TrimSpace(stackTrace)

		panic(fmt.Sprintf("\n%s\n%s", stackTrace, message))
	}
}

func pruneStack(fullStackTrace string, skip int) string {
	stack := strings.Split(fullStackTrace, "\n")
	if len(stack) > 1+2*skip {
		stack = stack[1+2*skip:]
	}

	srcBlacklist := []string{
		"testing.tRunner",
		"created by testing.RunTests",
	}
	srcBlacklistRE := regexp.MustCompile(strings.Join(srcBlacklist, "|"))

	suffix := regexp.MustCompile(` \+0x[0-9a-f]+$`)
	trim := func(s string) string {
		return suffix.ReplaceAllString(s, "")
	}

	prunedStack := []string{}
	for i := 0; i < len(stack)/2; i++ {
		if srcBlacklistRE.Match([]byte(stack[i*2])) {
			continue
		}

		prunedStack = append(prunedStack, trim(stack[i*2]))
		prunedStack = append(prunedStack, trim(stack[i*2+1]))
	}

	return strings.Join(prunedStack, "\n")
}
