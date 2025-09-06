package kubernetes

import (
	"context"
	"crypto/x509"
	"fmt"
	"log/slog"

	"github.com/open-policy-agent/cert-controller/pkg/rotator"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type CertManager struct {
	SecretName          string
	CertDir             string
	WebhookService      string
	K8sClusterDomain    string
	CAName              string
	CAOrganization      string
	MutatingWebhookName string
	Ready               chan struct{}
	Client              Client
}

// AddCertificateRotation registers all needed services to generate the certificates and patches needed resources with the caBundle.
func (cm *CertManager) AddCertificateRotation(ctx context.Context, mgr manager.Manager) error {
	namespace, err := getCurrentNamespace()
	if err != nil {
		return err
	}

	webhookRotators := []rotator.WebhookInfo{
		{
			Name: cm.MutatingWebhookName,
			Type: rotator.Mutating,
		},
	}

	secretAlreadyPresent, err := cm.Client.ensureSecret(namespace, cm.SecretName, ctx)
	if err != nil {
		return err
	}

	if secretAlreadyPresent {
		slog.Info("secret already present, skipping creation", "namespace", namespace, "name", cm.SecretName)
	} else {
		slog.Info("secret created or already present", "namespace", namespace, "name", cm.SecretName)
	}

	var extraDNSNames []string
	extraDNSNames = append(extraDNSNames, getDNSNames(namespace, cm.WebhookService, cm.K8sClusterDomain)...)

	slog.Info("preparing certificates rotation")

	err = rotator.AddRotator(mgr, &rotator.CertRotator{
		SecretKey: types.NamespacedName{
			Namespace: namespace,
			Name:      cm.SecretName,
		},
		CertDir:                cm.CertDir,
		CAName:                 cm.CAName,
		CAOrganization:         cm.CAOrganization,
		DNSName:                extraDNSNames[0],
		ExtraDNSNames:          extraDNSNames,
		IsReady:                cm.Ready,
		Webhooks:               webhookRotators,
		RestartOnSecretRefresh: true,
		RequireLeaderElection:  true,
		ExtKeyUsages: &[]x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to add certificates rotator: %w", err)
	} else {
		slog.Info("certificates rotator added to controller runtime")
		return nil
	}
}

// getDNSNames  creates all the possible DNS names for a given service.
func getDNSNames(namespace, service, k8sClusterDomain string) []string {
	return []string{
		service,
		fmt.Sprintf("%s.%s", service, namespace),
		fmt.Sprintf("%s.%s.svc", service, namespace),
		fmt.Sprintf("%s.%s.svc.%s", service, namespace, k8sClusterDomain),
	}
}
