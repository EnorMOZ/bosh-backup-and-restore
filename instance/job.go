package instance

import (
	"fmt"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/pkg/errors"
)

func NewJob(remoteRunner RemoteRunner, instanceIdentifier string, logger Logger, release string, jobScripts BackupAndRestoreScripts, metadata Metadata) Job {
	jobName := jobScripts[0].JobName()
	return Job{
		Logger:             logger,
		remoteRunner:       remoteRunner,
		instanceIdentifier: instanceIdentifier,
		name:               jobName,
		release:            release,
		metadata:           metadata,
		backupScript:       jobScripts.BackupOnly().firstOrBlank(),
		restoreScript:      jobScripts.RestoreOnly().firstOrBlank(),
		preBackupScript:    jobScripts.PreBackupLockOnly().firstOrBlank(),
		preRestoreScript:   jobScripts.PreRestoreLockOnly().firstOrBlank(),
		postBackupScript:   jobScripts.PostBackupUnlockOnly().firstOrBlank(),
		postRestoreScript:  jobScripts.SinglePostRestoreUnlockScript(),
	}
}

type Job struct {
	Logger             Logger
	name               string
	release            string
	metadata           Metadata
	backupScript       Script
	preBackupScript    Script
	postBackupScript   Script
	preRestoreScript   Script
	restoreScript      Script
	postRestoreScript  Script
	remoteRunner       RemoteRunner
	instanceIdentifier string
}

func (j Job) Name() string {
	return j.name
}

func (j Job) Release() string {
	return j.release
}

func (j Job) InstanceIdentifier() string {
	return j.instanceIdentifier
}

func (j Job) BackupArtifactName() string {
	return j.metadata.BackupName
}

func (j Job) RestoreArtifactName() string {
	return j.metadata.RestoreName
}

func (j Job) BackupArtifactDirectory() string {
	return fmt.Sprintf("%s/%s", orchestrator.ArtifactDirectory, j.backupArtifactOrJobName())
}

func (j Job) RestoreArtifactDirectory() string {
	return fmt.Sprintf("%s/%s", orchestrator.ArtifactDirectory, j.restoreArtifactOrJobName())
}

func (j Job) RestoreScript() Script {
	return j.restoreScript
}

func (j Job) HasBackup() bool {
	return j.backupScript != ""
}

func (j Job) HasRestore() bool {
	return j.RestoreScript() != ""
}

func (j Job) HasNamedBackupArtifact() bool {
	return j.metadata.BackupName != ""
}

func (j Job) HasNamedRestoreArtifact() bool {
	return j.metadata.RestoreName != ""
}

func (j Job) Backup() error {
	if j.backupScript != "" {
		j.Logger.Debug("bbr", "> %s", j.backupScript)
		j.Logger.Info("bbr", "Backing up %s on %s...", j.name, j.instanceIdentifier)

		err := j.remoteRunner.CreateDirectory(j.BackupArtifactDirectory())
		if err != nil {
			return err
		}

		env := artifactDirectoryVariables(j.BackupArtifactDirectory())
		_, err = j.remoteRunner.RunScriptWithEnv(string(j.backupScript), env)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf(
				"Error attempting to run backup for job %s on %s. Error: %s",
				j.Name(),
				j.instanceIdentifier,
				err.Error(),
			))
		}

		j.Logger.Info("bbr", "Done.")
	}

	return nil
}

func (j Job) PreBackupLock() error {
	if j.preBackupScript != "" {
		j.Logger.Debug("bbr", "> %s", j.preBackupScript)
		j.Logger.Info("bbr", "Locking %s on %s for backup...", j.name, j.instanceIdentifier)

		_, err := j.remoteRunner.RunScript(string(j.preBackupScript))
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf(
				"Error attempting to run pre-backup-lock for job %s on %s. Error: %s",
				j.Name(),
				j.instanceIdentifier,
				err.Error(),
			))
		}

		j.Logger.Info("bbr", "Done.")
	}

	return nil
}

func (j Job) PostBackupUnlock() error {
	if j.postBackupScript != "" {
		j.Logger.Debug("bbr", "> %s", j.postBackupScript)
		j.Logger.Info("bbr", "Unlocking %s on %s...", j.name, j.instanceIdentifier)

		_, err := j.remoteRunner.RunScript(string(j.postBackupScript))
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf(
				"Error attempting to run post-backup-unlock for job %s on %s. Error: %s",
				j.Name(),
				j.instanceIdentifier,
				err.Error(),
			))
		}

		j.Logger.Info("bbr", "Done.")
	}

	return nil
}

func (j Job) PreRestoreLock() error {
	if j.preRestoreScript != "" {
		j.Logger.Debug("bbr", "> %s", j.preRestoreScript)
		j.Logger.Info("bbr", "Locking %s on %s for restore...", j.name, j.instanceIdentifier)

		_, err := j.remoteRunner.RunScript(string(j.preRestoreScript))
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf(
				"Error attempting to run pre-restore-lock for job %s on %s. Error: %s",
				j.Name(),
				j.instanceIdentifier,
				err.Error(),
			))
		}

		j.Logger.Info("bbr", "Done.")
	}

	return nil
}

func (j Job) Restore() error {
	if j.restoreScript != "" {
		j.Logger.Debug("bbr", "> %s", j.restoreScript)
		j.Logger.Info("bbr", "Restoring %s on %s...", j.name, j.instanceIdentifier)

		env := artifactDirectoryVariables(j.RestoreArtifactDirectory())
		_, err := j.remoteRunner.RunScriptWithEnv(string(j.restoreScript), env)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf(
				"Error attempting to run restore for job %s on %s. Error: %s",
				j.Name(),
				j.instanceIdentifier,
				err.Error(),
			))
		}

		j.Logger.Info("bbr", "Done.")
	}

	return nil
}

func (j Job) PostRestoreUnlock() error {
	if j.postRestoreScript != "" {
		j.Logger.Debug("bbr", "> %s", j.postRestoreScript)
		j.Logger.Info("bbr", "Unlocking %s on %s...", j.name, j.instanceIdentifier)

		_, err := j.remoteRunner.RunScript(string(j.postRestoreScript))
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf(
				"Error attempting to run post-restore-unlock for job %s on %s. Error: %s",
				j.Name(),
				j.instanceIdentifier,
				err.Error(),
			))
		}

		j.Logger.Info("bbr", "Done.")
	}

	return nil
}

func (j Job) backupArtifactOrJobName() string {
	if j.HasNamedBackupArtifact() {
		return j.BackupArtifactName()
	}

	return j.name
}

func (j Job) restoreArtifactOrJobName() string {
	if j.HasNamedRestoreArtifact() {
		return j.RestoreArtifactName()
	}

	return j.name
}

func (j Job) handleErrs(jobName, label string, err error, exitCode int, stdout, stderr []byte) error {
	var foundErrors []error

	if err != nil {
		j.Logger.Error("bbr", fmt.Sprintf(
			"Error attempting to run %s script for job %s on %s. Error: %s",
			label,
			jobName,
			j.instanceIdentifier,
			err.Error(),
		))
		foundErrors = append(foundErrors, err)
	} else if exitCode != 0 {
		errorString := fmt.Sprintf(
			"%s script for job %s failed on %s.\nStdout: %s\nStderr: %s",
			label,
			jobName,
			j.instanceIdentifier,
			stdout,
			stderr,
		)

		foundErrors = append(foundErrors, errors.New(errorString))

		j.Logger.Error("bbr", errorString)
	}

	return orchestrator.ConvertErrors(foundErrors)
}

func (j Job) BackupShouldBeLockedBefore() []orchestrator.JobSpecifier {
	jobSpecifiers := []orchestrator.JobSpecifier{}

	for _, lockBefore := range j.metadata.BackupShouldBeLockedBefore {
		jobSpecifiers = append(jobSpecifiers, orchestrator.JobSpecifier{
			Name: lockBefore.JobName, Release: lockBefore.Release,
		})
	}

	return jobSpecifiers
}

func (j Job) RestoreShouldBeLockedBefore() []orchestrator.JobSpecifier {
	jobSpecifiers := []orchestrator.JobSpecifier{}

	for _, lockBefore := range j.metadata.RestoreShouldBeLockedBefore {
		jobSpecifiers = append(jobSpecifiers, orchestrator.JobSpecifier{
			Name: lockBefore.JobName, Release: lockBefore.Release,
		})
	}

	return jobSpecifiers
}
