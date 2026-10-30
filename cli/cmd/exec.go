package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"claimctl-cli/pkg/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type execOptions struct {
	claim     api.ClaimRequest
	heartbeat time.Duration
	timeout   int
	poll      int
	grace     time.Duration
	key       string
}

func reservationDeadline(r *api.Reservation) time.Time {
	var deadline time.Time
	for _, value := range []*time.Time{r.LeaseExpiresAt, r.ScheduledEndTime, r.HardEndTime} {
		if value != nil && (deadline.IsZero() || value.Before(deadline)) {
			deadline = *value
		}
	}
	return deadline
}

func maintainLease(ctx context.Context, client *api.Client, r *api.Reservation, interval time.Duration, fail func(error)) {
	for {
		deadline := reservationDeadline(r)
		if r.LeaseExpiresAt == nil || deadline.IsZero() {
			fail(fmt.Errorf("server did not return a finite lease"))
			return
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			fail(fmt.Errorf("reservation deadline elapsed"))
			return
		}
		timer := time.NewTimer(min(interval, remaining))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		if !time.Now().Before(deadline) {
			fail(fmt.Errorf("reservation deadline elapsed"))
			return
		}
		renewCtx, cancel := context.WithDeadline(ctx, deadline)
		renewClient := *client
		renewClient.Context = renewCtx
		next, err := renewClient.Renew(r.ID, "")
		cancel()
		if err != nil {
			if ctx.Err() == nil {
				fail(fmt.Errorf("lease renewal failed: %w", err))
			}
			return
		}
		r = next
	}
}

func runClaimedCommand(cmd *cobra.Command, args []string, opts execOptions) (resultErr error) {
	if viper.GetBool("json") {
		return fmt.Errorf("exec streams child output and cannot be combined with json")
	}
	if len(args) == 0 {
		return fmt.Errorf("provide a command after --")
	}
	if opts.heartbeat <= 0 || opts.timeout <= 0 || opts.poll <= 0 || opts.grace <= 0 {
		return fmt.Errorf("heartbeat, timeout, poll interval and kill grace must be positive")
	}
	lease, err := time.ParseDuration(opts.claim.LeaseDuration)
	if err != nil || lease < time.Second || opts.heartbeat >= lease/2 {
		return fmt.Errorf("lease duration must be at least one second and heartbeat less than half the lease")
	}
	client, err := newClient(cmd)
	if err != nil {
		return err
	}
	workCtx, cancel := context.WithCancelCause(cmd.Context())
	defer cancel(nil)
	client.Context = workCtx
	r, err := client.Claim(opts.claim, opts.key)
	if err != nil {
		return err
	}
	reservationID := r.ID
	activated := r.Status == "active"
	needsCleanup := true
	defer func() {
		if !needsCleanup {
			return
		}
		cleanupCtx, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		cleanup := *client
		cleanup.Context = cleanupCtx
		var cleanupErr error
		if activated {
			cleanupErr = cleanup.CompleteReservation(reservationID)
		} else {
			cleanupErr = cleanup.CancelReservation(reservationID)
		}
		if cleanupErr != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("cleanup reservation %s: %w", reservationID, cleanupErr))
		}
	}()
	current, err := client.GetReservation(reservationID)
	if err != nil {
		return err
	}
	r = current
	activated = r.Status == "active"
	if !activated && r.Status != "pending" && r.Status != "queued" {
		needsCleanup = false
		return fmt.Errorf("reservation is %s", r.Status)
	}
	if r.LeaseExpiresAt == nil || r.HardEndTime == nil || reservationDeadline(r).IsZero() {
		return fmt.Errorf("server did not return finite lease deadlines")
	}
	if !time.Now().Before(reservationDeadline(r)) {
		return fmt.Errorf("reservation deadline elapsed")
	}
	startMonitor := func(reservation *api.Reservation) (context.CancelFunc, <-chan struct{}) {
		monitorCtx, stop := context.WithCancel(workCtx)
		done := make(chan struct{})
		go func() {
			defer close(done)
			maintainLease(monitorCtx, client, reservation, opts.heartbeat, func(err error) { cancel(err) })
		}()
		return stop, done
	}
	stopMonitor, monitorDone := startMonitor(r)
	defer func() { stopMonitor(); <-monitorDone }()
	if !activated {
		if err = client.WaitForReservation(r.ID, opts.timeout, opts.poll, nil); err != nil {
			if cause := context.Cause(workCtx); cause != nil {
				return cause
			}
			return err
		}
		activated = true
		current, err = client.GetReservation(reservationID)
		if err != nil {
			return err
		}
		if current.Status != "active" || !time.Now().Before(reservationDeadline(current)) {
			return fmt.Errorf("reservation is no longer live and active")
		}
		stopMonitor()
		<-monitorDone
		r = current
		stopMonitor, monitorDone = startMonitor(r)
	}
	if workCtx.Err() != nil {
		return context.Cause(workCtx)
	}
	process := exec.Command(args[0], args[1:]...)
	process.Stdin = cmd.InOrStdin()
	process.Stdout = cmd.OutOrStdout()
	process.Stderr = cmd.ErrOrStderr()
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "CLAIMCTL_TOKEN=") && !strings.HasPrefix(entry, "CLAIMCTL_TOKEN_FILE=") && !strings.HasPrefix(entry, "CLAIMCTL_RESERVATION_ID=") && !strings.HasPrefix(entry, "CLAIMCTL_RESOURCE_ID=") {
			process.Env = append(process.Env, entry)
		}
	}
	process.Env = append(process.Env, "CLAIMCTL_RESERVATION_ID="+r.ID, "CLAIMCTL_RESOURCE_ID="+r.ResourceID)
	restoreTerminal := prepareProcess(process)
	defer restoreTerminal()
	signals := make(chan os.Signal, 2)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	if err = process.Start(); err != nil {
		return err
	}
	defer killProcess(process)
	done := make(chan error, 1)
	go func() { done <- process.Wait() }()
	var stopReason error
	select {
	case err = <-done:
		return commandExitError(err)
	case sig := <-signals:
		signalProcess(process, sig)
		stopReason = &CLIError{Message: "command interrupted", ExitCode: signalExitCode(sig)}
	case <-workCtx.Done():
		select {
		case sig := <-signals:
			signalProcess(process, sig)
			stopReason = &CLIError{Message: "command interrupted", ExitCode: signalExitCode(sig)}
		default:
			stopReason = context.Cause(workCtx)
			var interrupted *CLIError
			if errors.As(stopReason, &interrupted) && errors.Is(stopReason, context.Canceled) && interrupted.ExitCode >= 128 {
				signalProcess(process, syscall.Signal(interrupted.ExitCode-128))
			} else {
				signalProcess(process, syscall.SIGTERM)
			}
		}
	}
	timer := time.NewTimer(opts.grace)
	defer timer.Stop()
	select {
	case <-done:
	case <-timer.C:
		killProcess(process)
		<-done
	}
	return stopReason
}

func commandExitError(err error) error {
	if err == nil {
		return nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return &CLIError{Message: "child command failed", ExitCode: processExitCode(exit), Err: err}
	}
	return err
}

func init() {
	opts := execOptions{}
	command := &cobra.Command{Use: "exec [resource-id] -- command [args...]", Short: "Claim, heartbeat and release a resource around a child command", RunE: func(cmd *cobra.Command, args []string) error {
		split := cmd.ArgsLenAtDash()
		if split < 0 {
			return fmt.Errorf("separate the child command with --")
		}
		if split > 1 {
			return fmt.Errorf("provide at most one resource ID before --")
		}
		request := opts
		if split == 1 {
			request.claim.ResourceID = args[0]
		}
		return runClaimedCommand(cmd, args[split:], request)
	}}
	flags := command.Flags()
	flags.StringVar(&opts.claim.Name, "name", "", "Exact resource name")
	flags.StringVar(&opts.claim.Type, "type", "", "Resource type")
	flags.StringVar(&opts.claim.LabelExpr, "label-expr", "", "Label expression")
	flags.StringVar(&opts.claim.Duration, "duration", "", "Finite hold duration (defaults to server policy)")
	flags.StringVar(&opts.claim.LeaseDuration, "lease-duration", "30s", "Heartbeat lease duration")
	flags.BoolVar(&opts.claim.NoQueue, "no-queue", false, "Fail atomically without queuing")
	flags.BoolVar(&opts.claim.RequireHealthy, "require-healthy", false, "Require persisted healthy status")
	flags.DurationVar(&opts.heartbeat, "heartbeat-interval", 10*time.Second, "Heartbeat interval, less than half the lease")
	flags.IntVar(&opts.timeout, "timeout", 300, "Activation timeout in seconds")
	flags.IntVar(&opts.poll, "poll-interval", 5, "Activation polling interval in seconds")
	flags.DurationVar(&opts.grace, "kill-grace", 5*time.Second, "Grace before killing a stopped process group")
	flags.StringVar(&opts.key, "idempotency-key", "", "Creation retry key (random by default)")
	rootCmd.AddCommand(command)
}
