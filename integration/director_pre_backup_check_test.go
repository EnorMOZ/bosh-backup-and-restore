package integration

import (
	"io/ioutil"
	"os"

	"github.com/pivotal-cf/bosh-backup-and-restore/testcluster"

	"github.com/onsi/gomega/gexec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pre-backup checks", func() {
	var backupWorkspace string
	var session *gexec.Session
	var directorIP string

	BeforeEach(func() {
		var err error
		backupWorkspace, err = ioutil.TempDir(".", "backup-workspace-")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(backupWorkspace)).To(Succeed())
	})

	JustBeforeEach(func() {
		session = runBinary(
			backupWorkspace,
			[]string{"BOSH_CLIENT_SECRET=admin"},
			"director",
			"--name", "my-director",
			"--host", directorIP,
			"--username", "foobar",
			"--private-key-path", pathToPrivateKeyFile,
			"pre-backup-check",
		)
	})

	Context("When there is a director instance", func() {
		Context("and there is a backup script", func() {
			var directorInstance *testcluster.Instance

			BeforeEach(func() {
				directorInstance = testcluster.NewInstance()
				directorInstance.CreateUser("foobar", readFile(pathToPublicKeyFile))
				By("creating a dummy backup script")
				directorInstance.CreateScript("/var/vcap/jobs/redis/bin/bbr/backup", `#!/usr/bin/env sh
set -u
printf "backupcontent1" > $BBR_ARTIFACT_DIRECTORY/backupdump1
printf "backupcontent2" > $BBR_ARTIFACT_DIRECTORY/backupdump2
`)
				directorIP = directorInstance.Address()

			})

			AfterEach(func() {
				directorInstance.DieInBackground()
			})

			It("exits zero", func() {
				Expect(session.ExitCode()).To(BeZero())
			})

			It("outputs a log message saying the director instance can be backed up", func() {
				Expect(string(session.Out.Contents())).To(ContainSubstring("Director can be backed up."))
			})

			Context("but the backup artifact directory already exists", func() {
				BeforeEach(func() {
					directorInstance.CreateDir("/var/vcap/store/bbr-backup")
				})

				It("exits non-zero", func() {
					Expect(session.ExitCode()).NotTo(BeZero())
				})

				It("outputs a log message saying the director instance cannot be backed up", func() {
					Expect(string(session.Out.Contents())).To(ContainSubstring("Director cannot be backed up."))
					Expect(string(session.Err.Contents())).To(ContainSubstring("Directory '/var/vcap/store/bbr-backup' already exists on instance bosh/0"))
				})

				It("does not delete the existing artifact directory", func() {
					Expect(directorInstance.FileExists("/var/vcap/store/bbr-backup")).To(BeTrue())
				})
			})
		})

		Context("if there are no backup scripts", func() {
			var directorInstance *testcluster.Instance

			BeforeEach(func() {
				directorInstance = testcluster.NewInstance()
				directorInstance.CreateUser("foobar", readFile(pathToPublicKeyFile))

				directorInstance.CreateFiles(
					"/var/vcap/jobs/redis/bin/not-a-backup-script",
				)
				directorIP = directorInstance.Address()
			})

			AfterEach(func() {
				directorInstance.DieInBackground()
			})

			It("returns exit code 1", func() {
				Expect(session.ExitCode()).To(Equal(1))
			})

			It("prints an error", func() {
				Expect(string(session.Out.Contents())).To(ContainSubstring("Director cannot be backed up."))
				Expect(string(session.Err.Contents())).To(ContainSubstring("Deployment 'my-director' has no backup scripts"))
			})
		})
	})

	Context("When the director does not resolve", func() {
		BeforeEach(func() {
			directorIP = "no:22"
		})

		It("returns exit code 1", func() {
			Expect(session.ExitCode()).To(Equal(1))
		})

		It("prints an error", func() {
			Expect(string(session.Err.Contents())).To(ContainSubstring("no such host"))
		})
	})
})

func readFile(fileName string) string {
	contents, err := ioutil.ReadFile(fileName)
	Expect(err).NotTo(HaveOccurred())
	return string(contents)
}
