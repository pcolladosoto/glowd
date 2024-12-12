//go:build linux && cgo

package ebpf

import (
	_ "embed"
	"fmt"
	"log/slog"
	"syscall"
	"time"
	"unsafe"

	"math/rand"

	bpf "github.com/aquasecurity/libbpfgo"
	glowdTypes "github.com/scitags/flowd-go/types"
)

// Please note this example has been basically plundered from:
//   https://github.com/libbpf/libbpf-bootstrap/blob/4a567f229efe8fc79ee1a2249569eb6b9c02ad1b/examples/c/tc.c
//   https://github.com/aquasecurity/libbpfgo/blob/282d44353ac28b015afb469d378e9d178afd3304/selftest/tc/main.go

const (
	PROG_NAME string = "marker"
	MAP_NAME  string = "flowLabels"
)

var (
	//go:embed progs/marker-flow-label.bpf.o
	flowLabelBPFProg []byte

	//go:embed progs/marker-flow-label-dbg.bpf.o
	flowLabelDebugBPFProg []byte

	//go:embed progs/marker-hbh-header.bpf.o
	hopByHopHeaderBPFProg []byte

	//go:embed progs/marker-hbh-header-dbg.bpf.o
	hopByHopHeaderDebugBPFProg []byte

	//go:embed progs/marker-hbh-do-headers.bpf.o
	hopByHopDestHeaderBPFProg []byte

	//go:embed progs/marker-hbh-do-headers-dbg.bpf.o
	hopByHopDestHeaderDebugBPFProg []byte

	logLevelTranslation = map[slog.Level]int{
		slog.LevelDebug: bpf.LibbpfDebugLevel,
		slog.LevelInfo:  bpf.LibbpfInfoLevel,
		slog.LevelWarn:  bpf.LibbpfWarnLevel,
	}
)

type flowFourTuple struct {
	IPv6Hi  uint64
	IPv6Lo  uint64
	DstPort uint16
	SrcPort uint16
}

type EbpfBackend struct {
	module  *bpf.Module
	hook    *bpf.TcHook
	flowMap *bpf.BPFMap

	tcOpts bpf.TcOpts

	rGen *rand.Rand

	TargetInterface string
	RemoveQdisc     bool
	ProgramPath     string
	MarkingStrategy MarkingStrategy
	DebugMode       bool
}

func (b *EbpfBackend) String() string {
	return "eBPF"
}

func (b *EbpfBackend) Init() error {
	slog.Debug("initialising the eBPF backend")

	// Setup the logging from libbpf
	b.SetupLogging()

	// Create the BPF module
	bpfModule, err := bpf.NewModuleFromBuffer(b.chooseBPFProgram(), "glowd")
	// bpfModule, err := bpf.NewModuleFromFile("main.bpf.o")
	if err != nil {
		return fmt.Errorf("error creating the BPF module: %w", err)
	}
	b.module = bpfModule

	// Try to load the module's object
	if err := b.module.BPFLoadObject(); err != nil {
		return fmt.Errorf("error loading the BPF object: %w", err)
	}

	// Create the TC Hook on which to place the eBPF program
	b.hook = b.module.TcHookInit()
	if err := b.hook.SetInterfaceByName(b.TargetInterface); err != nil {
		return fmt.Errorf("failed to set TC hook on interface %s: %w", b.TargetInterface, err)
	}

	// Placce the hook in the packet egress chain
	b.hook.SetAttachPoint(bpf.BPFTcEgress)
	if err := b.hook.Create(); err != nil {
		if errno, ok := err.(syscall.Errno); ok && errno != syscall.EEXIST {
			slog.Debug("error creating the tc hook", "err", err)
		}
	}

	// Recover the specific program (i.e. function) we'll be attaching
	tcProg, err := b.module.GetProgram(PROG_NAME)
	if tcProg == nil || err != nil {
		return fmt.Errorf("couldn't find the target program %s: %w", PROG_NAME, err)
	}

	// Prepare the options for the hook
	// https://elixir.bootlin.com/linux/v6.8.4/source/tools/testing/selftests/bpf/prog_tests/tc_bpf.c#L26
	var tcOpts bpf.TcOpts
	tcOpts.ProgFd = int(tcProg.FileDescriptor())
	tcOpts.Handle = 1
	tcOpts.Priority = 1

	// Attach the program!
	if err := b.hook.Attach(&tcOpts); err != nil {
		return fmt.Errorf("couldn't attach the ebpf program: %w", err)
	}

	// Get a reference to the map so that we're ready when running
	bpfMap, err := b.module.GetMap(MAP_NAME)
	if err != nil {
		return fmt.Errorf("error getting the ebpf map: %w", err)
	}
	b.flowMap = bpfMap

	// Test we can get the program back to check everything's okay
	tcOpts.ProgFd = 0
	tcOpts.ProgId = 0
	if err := b.hook.Query(&tcOpts); err != nil {
		return fmt.Errorf("error querying for the ebpf program: %w", err)
	}
	if tcOpts.Handle != 1 {
		return fmt.Errorf("recovered handle %d is different than expected (i.e. 1)", tcOpts.Handle)
	}
	// Get a hold of the tcOpts once all the operations are done: it can be implicitly changed as
	// we're passing it by reference!
	b.tcOpts = tcOpts

	// Initialise the random number generator
	b.rGen = rand.New(rand.NewSource(time.Now().UnixNano()))

	return nil
}

func (b *EbpfBackend) Run(done <-chan struct{}, inChan <-chan glowdTypes.FlowID) {
	slog.Debug("running the ebpf backend")

	for {
		select {
		case flowID, ok := <-inChan:
			if !ok {
				slog.Warn("somebody closed the input channel!")
				return
			}
			slog.Debug("got a flowID", "flowID", flowID)

			rawDstIPHi, rawDstIPLo := extractHalves(flowID.Dst.IP)
			flowHash := flowFourTuple{
				IPv6Hi:  rawDstIPHi,
				IPv6Lo:  rawDstIPLo,
				DstPort: flowID.Dst.Port,
				SrcPort: flowID.Src.Port,
			}

			slog.Debug("flowID.Dst.IP", "rawDstIP", fmt.Sprintf("%+v", []byte(flowID.Dst.IP)),
				"rawDstIPHi", rawDstIPHi, "rawDstIPLo", rawDstIPLo)

			switch flowID.State {
			case glowdTypes.START:
				flowTag := b.genFlowTag(flowID.Experiment, flowID.Activity)

				flowHashPtr := unsafe.Pointer(&flowHash)
				flowTagPtr := unsafe.Pointer(&flowTag)
				if err := b.flowMap.Update(flowHashPtr, flowTagPtr); err != nil {
					slog.Error("error inserting map value", "err", err, "flowHash", flowHash, "flowTag", flowTag)
					continue
				}
				slog.Debug("inserted map value", "flowHash", flowHash, "flowTag", flowTag)
			case glowdTypes.END:
				flowHashPtr := unsafe.Pointer(&flowHash)
				if err := b.flowMap.DeleteKey(flowHashPtr); err != nil {
					slog.Error("error deleting map key", "err", err, "flowHash", flowHash)
					continue
				}
				slog.Debug("deleted map value", "flowHash", flowHash)
			default:
				slog.Error("wrong flow state made it here", "flowID.State", flowID.State)
			}
		case <-done:
			slog.Debug("cleanly exiting the ebpf backend")
			return
		}
	}
}

func (b *EbpfBackend) Cleanup() error {
	slog.Debug("cleaning up the ebpf backend")

	// This explicit checks aren't really needed; it's simply left here for emphasis!
	if b.hook != nil {
		// Detach the program. Note ProgFd and ProgId must be set to 0 or the detachment
		// won't work...
		localTcOpts := b.tcOpts
		localTcOpts.ProgFd = 0
		localTcOpts.ProgId = 0

		slog.Debug("detaching the tc hook")
		if err := b.hook.Detach(&localTcOpts); err != nil {
			slog.Error("error detaching the ebpf hook", "err", err)
		}

		if b.RemoveQdisc {
			slog.Debug("removing the backing qdisc")
			// Explicitly ask for the backing QDisc to be destroyed.
			// See https://patchwork.kernel.org/project/netdevbpf/patch/20210428162553.719588-3-memxor@gmail.com/
			b.hook.SetAttachPoint(bpf.BPFTcEgress | bpf.BPFTcIngress)
		}
		slog.Debug("destroying the tc hook")
		if err := b.hook.Destroy(); err != nil {
			slog.Warn("error destroying the hook", "err", err)
		}
	}

	// This explicit checks aren't really needed; it's simply left here for emphasis!
	if b.module != nil {
		slog.Debug("closing the eBPF module")
		b.module.Close()
	}

	return nil
}
