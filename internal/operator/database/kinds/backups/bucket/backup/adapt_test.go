package backup

import (
	"github.com/caos/orbos/mntr"
	"github.com/caos/orbos/pkg/kubernetes"
	kubernetesmock "github.com/caos/orbos/pkg/kubernetes/mock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	macherrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"testing"
)

func TestBackup_AdaptInstantBackup1(t *testing.T) {
	client := kubernetesmock.NewMockClientInt(gomock.NewController(t))

	features := []string{Instant}
	monitor := mntr.Monitor{}
	namespace := "testNs"
	labels := map[string]string{"test": "test"}
	databases := []string{"testDb"}
	bucketName := "testBucket"
	cron := "testCron"
	timestamp := "test"
	nodeselector := map[string]string{"test": "test"}
	tolerations := []corev1.Toleration{
		{Key: "testKey", Operator: "testOp"}}
	backupName := "testName"
	version := "testVersion"
	secretKey := "testKey"
	secretName := "testSecretName"
	jobName := cronJobNamePrefix + backupName

	checkDBReady := func(k8sClient kubernetes.ClientInt) error {
		return nil
	}

	jobDef := getJob(
		namespace,
		labels,
		jobName,
		getJobSpecDef(
			nodeselector,
			tolerations,
			secretName,
			secretKey,
			backupName,
			version,
			getBackupCommand(
				timestamp,
				databases,
				bucketName,
				backupName,
			),
		),
	)

	client.EXPECT().ApplyJob(jobDef).Times(1).Return(nil)
	client.EXPECT().GetJob(jobDef.Namespace, jobDef.Name).Times(1).Return(nil, macherrs.NewNotFound(schema.GroupResource{"batch", "jobs"}, jobName))
	client.EXPECT().WaitUntilJobCompleted(jobDef.Namespace, jobDef.Name, timeout).Times(1).Return(nil)
	client.EXPECT().DeleteJob(jobDef.Namespace, jobDef.Name).Times(1).Return(nil)

	query, _, err := AdaptFunc(
		monitor,
		backupName,
		namespace,
		labels,
		databases,
		checkDBReady,
		bucketName,
		cron,
		secretName,
		secretKey,
		timestamp,
		nodeselector,
		tolerations,
		features,
		version,
	)

	assert.NoError(t, err)
	queried := map[string]interface{}{}
	ensure, err := query(client, queried)
	assert.NoError(t, err)
	assert.NoError(t, ensure(client))
}

func TestBackup_AdaptInstantBackup2(t *testing.T) {
	client := kubernetesmock.NewMockClientInt(gomock.NewController(t))

	features := []string{Instant}
	monitor := mntr.Monitor{}
	namespace := "testNs2"
	labels := map[string]string{"test2": "test2"}
	databases := []string{"testDb2"}
	bucketName := "testBucket2"
	cron := "testCron2"
	timestamp := "test2"
	nodeselector := map[string]string{"test2": "test2"}
	tolerations := []corev1.Toleration{
		{Key: "testKey2", Operator: "testOp2"}}
	backupName := "testName2"
	version := "testVersion2"
	secretKey := "testKey2"
	secretName := "testSecretName2"
	jobName := GetJobName(backupName)

	checkDBReady := func(k8sClient kubernetes.ClientInt) error {
		return nil
	}

	jobDef := getJob(
		namespace,
		labels,
		jobName,
		getJobSpecDef(
			nodeselector,
			tolerations,
			secretName,
			secretKey,
			backupName,
			version,
			getBackupCommand(
				timestamp,
				databases,
				bucketName,
				backupName,
			),
		),
	)

	client.EXPECT().ApplyJob(jobDef).Times(1).Return(nil)
	client.EXPECT().GetJob(jobDef.Namespace, jobDef.Name).Times(1).Return(nil, macherrs.NewNotFound(schema.GroupResource{"batch", "jobs"}, jobName))
	client.EXPECT().WaitUntilJobCompleted(jobDef.Namespace, jobDef.Name, timeout).Times(1).Return(nil)
	client.EXPECT().DeleteJob(jobDef.Namespace, jobDef.Name).Times(1).Return(nil)

	query, _, err := AdaptFunc(
		monitor,
		backupName,
		namespace,
		labels,
		databases,
		checkDBReady,
		bucketName,
		cron,
		secretName,
		secretKey,
		timestamp,
		nodeselector,
		tolerations,
		features,
		version,
	)

	assert.NoError(t, err)
	queried := map[string]interface{}{}
	ensure, err := query(client, queried)
	assert.NoError(t, err)
	assert.NoError(t, ensure(client))
}

func TestBackup_AdaptBackup1(t *testing.T) {
	client := kubernetesmock.NewMockClientInt(gomock.NewController(t))

	features := []string{Normal}
	monitor := mntr.Monitor{}
	namespace := "testNs"
	labels := map[string]string{"test": "test"}
	databases := []string{"testDb"}
	bucketName := "testBucket"
	cron := "testCron"
	timestamp := "test"
	nodeselector := map[string]string{"test": "test"}
	tolerations := []corev1.Toleration{
		{Key: "testKey", Operator: "testOp"}}
	backupName := "testName"
	version := "testVersion"
	secretKey := "testKey"
	secretName := "testSecretName"
	jobName := GetJobName(backupName)

	checkDBReady := func(k8sClient kubernetes.ClientInt) error {
		return nil
	}

	jobDef := getCronJob(
		namespace,
		labels,
		jobName,
		cron,
		getJobSpecDef(
			nodeselector,
			tolerations,
			secretName,
			secretKey,
			backupName,
			version,
			getBackupCommand(
				timestamp,
				databases,
				bucketName,
				backupName,
			),
		),
	)

	client.EXPECT().ApplyCronJob(jobDef).Times(1).Return(nil)

	query, _, err := AdaptFunc(
		monitor,
		backupName,
		namespace,
		labels,
		databases,
		checkDBReady,
		bucketName,
		cron,
		secretName,
		secretKey,
		timestamp,
		nodeselector,
		tolerations,
		features,
		version,
	)

	assert.NoError(t, err)
	queried := map[string]interface{}{}
	ensure, err := query(client, queried)
	assert.NoError(t, err)
	assert.NoError(t, ensure(client))
}

func TestBackup_AdaptBackup2(t *testing.T) {
	client := kubernetesmock.NewMockClientInt(gomock.NewController(t))

	features := []string{Normal}
	monitor := mntr.Monitor{}
	namespace := "testNs2"
	labels := map[string]string{"test2": "test2"}
	databases := []string{"testDb2"}
	bucketName := "testBucket2"
	cron := "testCron2"
	timestamp := "test2"
	nodeselector := map[string]string{"test2": "test2"}
	tolerations := []corev1.Toleration{
		{Key: "testKey2", Operator: "testOp2"}}
	backupName := "testName2"
	version := "testVersion2"
	secretKey := "testKey2"
	secretName := "testSecretName2"
	jobName := GetJobName(backupName)

	checkDBReady := func(k8sClient kubernetes.ClientInt) error {
		return nil
	}

	jobDef := getCronJob(
		namespace,
		labels,
		jobName,
		cron,
		getJobSpecDef(
			nodeselector,
			tolerations,
			secretName,
			secretKey,
			backupName,
			version,
			getBackupCommand(
				timestamp,
				databases,
				bucketName,
				backupName,
			),
		),
	)

	client.EXPECT().ApplyCronJob(jobDef).Times(1).Return(nil)

	query, _, err := AdaptFunc(
		monitor,
		backupName,
		namespace,
		labels,
		databases,
		checkDBReady,
		bucketName,
		cron,
		secretName,
		secretKey,
		timestamp,
		nodeselector,
		tolerations,
		features,
		version,
	)

	assert.NoError(t, err)
	queried := map[string]interface{}{}
	ensure, err := query(client, queried)
	assert.NoError(t, err)
	assert.NoError(t, ensure(client))
}