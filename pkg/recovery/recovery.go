package recovery

import (
	"fmt"
	"time"

	"github.com/appscode/go/log"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/client/clientset/versioned/scheme"
	cs "github.com/appscode/stash/client/clientset/versioned/typed/stash/v1alpha1"
	stash_util "github.com/appscode/stash/client/clientset/versioned/typed/stash/v1alpha1/util"
	"github.com/appscode/stash/pkg/cli"
	"github.com/appscode/stash/pkg/eventer"
	"github.com/appscode/stash/pkg/util"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/reference"
)

type Controller struct {
	k8sClient    kubernetes.Interface
	stashClient  cs.StashV1alpha1Interface
	namespace    string
	recoveryName string
}

const (
	RecoveryEventComponent = "stash-recovery"
)

func New(k8sClient kubernetes.Interface, stashClient cs.StashV1alpha1Interface, namespace, name string) *Controller {
	return &Controller{
		k8sClient:    k8sClient,
		stashClient:  stashClient,
		namespace:    namespace,
		recoveryName: name,
	}
}

func (c *Controller) Run() {
	recovery, err := c.stashClient.Recoveries(c.namespace).Get(c.recoveryName, metav1.GetOptions{})
	if err != nil {
		log.Errorln(err)
		return
	}

	if err = recovery.IsValid(); err != nil {
		log.Errorf("Failed to validate recovery %s, reason: %s\n", recovery.Name, err)
		stash_util.SetRecoveryStatusPhase(c.stashClient, recovery, api.RecoveryFailed)
		ref, rerr := reference.GetReference(scheme.Scheme, recovery)
		if rerr == nil {
			eventer.CreateEventWithLog(
				c.k8sClient,
				RecoveryEventComponent,
				ref,
				core.EventTypeWarning,
				eventer.EventReasonFailedToRecover,
				fmt.Sprintf("Failed to validate recovery %s, reason: %s", recovery.Name, err),
			)
		}
		return
	}

	if err = c.RecoverOrErr(recovery); err != nil {
		log.Errorf("Failed to complete recovery %s, reason: %s\n", recovery.Name, err)
		stash_util.SetRecoveryStatusPhase(c.stashClient, recovery, api.RecoveryFailed)
		ref, rerr := reference.GetReference(scheme.Scheme, recovery)
		if rerr == nil {
			eventer.CreateEventWithLog(
				c.k8sClient,
				RecoveryEventComponent,
				ref,
				core.EventTypeWarning,
				eventer.EventReasonFailedToRecover,
				fmt.Sprintf("Failed to complete recovery %s, reason: %s", recovery.Name, err),
			)
		}
		return
	}

	log.Infof("Recovery %s succeeded\n", recovery.Name)
	stash_util.SetRecoveryStatusPhase(c.stashClient, recovery, api.RecoverySucceeded) // TODO: status.Stats
	ref, rerr := reference.GetReference(scheme.Scheme, recovery)
	if rerr == nil {
		eventer.CreateEventWithLog(
			c.k8sClient,
			RecoveryEventComponent,
			ref,
			core.EventTypeNormal,
			eventer.EventReasonSuccessfulRecovery,
			fmt.Sprintf("Recovery %s succeeded", recovery.Name),
		)
	}
}

func (c *Controller) RecoverOrErr(recovery *api.Recovery) error {
	repository, err := c.stashClient.Repositories(recovery.Namespace).Get(recovery.Spec.Repository, metav1.GetOptions{})
	if err != nil {
		return err
	}
	repoLabelData, err := util.ExtractDataFromRepositoryLabel(repository.Labels)
	if err != nil {
		return err
	}
	secret, err := c.k8sClient.CoreV1().Secrets(c.namespace).Get(repository.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	workload := &api.LocalTypedReference{
		Kind: repoLabelData.WorkloadKind,
		Name: repoLabelData.WorkloadName,
	}
	hostname, smartPrefix, err := workload.HostnamePrefix(repoLabelData.PodName, repoLabelData.NodeName)
	if err != nil {
		return err
	}

	backend := util.FixBackendPrefix(repository.Spec.Backend.DeepCopy(), smartPrefix)

	cli := cli.New("/tmp", false, hostname)
	if _, err = cli.SetupEnv(*backend, secret, smartPrefix); err != nil {
		return err
	}

	snapshotID := ""
	if recovery.Spec.Snapshot != "" {
		_, snapshotID, err = util.GetRepoNameAndSnapshotID(recovery.Spec.Snapshot)
		if err != nil {
			return err
		}
	}

	var errRec error
	for _, path := range recovery.Spec.Paths {
		d, err := c.measure(cli.Restore, path, hostname, snapshotID)
		if err != nil {
			errRec = err
			ref, rerr := reference.GetReference(scheme.Scheme, recovery)
			if rerr == nil {
				eventer.CreateEventWithLog(
					c.k8sClient,
					RecoveryEventComponent,
					ref,
					core.EventTypeWarning,
					eventer.EventReasonFailedToRecover,
					fmt.Sprintf("failed to recover FileGroup %s, reason: %v", path, err),
				)
			}
			stash_util.SetRecoveryStats(c.stashClient, recovery, path, d, api.RecoveryFailed)
		} else {
			stash_util.SetRecoveryStats(c.stashClient, recovery, path, d, api.RecoverySucceeded)
		}
	}

	return errRec
}

func (c *Controller) measure(f func(string, string, string) error, path, host, snapshotID string) (time.Duration, error) {
	startTime := time.Now()
	err := f(path, host, snapshotID)
	return time.Now().Sub(startTime), err
}
