package ese

import (
	"fmt"
	"os/exec"
	"runtime"
	"testing"

	"github.com/alecthomas/assert"
	"github.com/sebdah/goldie"
	"github.com/stretchr/testify/suite"
)

type ESETestSuite struct {
	suite.Suite
	binary string
}

func (self *ESETestSuite) SetupTest() {
	self.binary = "./eseparser"
	if runtime.GOOS == "windows" {
		self.binary += ".exe"
	}
}

// User Access Logs have some interesting columns types:
// * GUID
// * DateTime seem to be encoded in a different way - a uint64 windows
//   file time.
func (self *ESETestSuite) TestUAL() {
	cmdline := []string{
		"dump", "--limit", "2",
		"testdata/Sample_UAL/HyperV-PC/Current.mdb", "CLIENTS",
	}
	cmd := exec.Command(self.binary, cmdline...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(out))
	}
	assert.NoError(self.T(), err)

	fixture_name := "UAL_CLIENTS"
	goldie.Assert(self.T(), fixture_name, out)
}

func (self *ESETestSuite) TestSRUM() {
	cmdline := []string{
		"dump", "--limit", "2",
		"testdata/SRUM/SRUDB.dat", "{D10CA2FE-6FCF-4F6D-848E-B2E99266FA86}",
	}
	cmd := exec.Command(self.binary, cmdline...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		assert.NoError(self.T(), err)
	}
	fixture_name := "SRUM-D10CA2FE-6FCF-4F6D-848E-B2E99266FA86"
	goldie.Assert(self.T(), fixture_name, out)
}

func TestESE(t *testing.T) {
	suite.Run(t, &ESETestSuite{})
}
