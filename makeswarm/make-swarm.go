package makeswarm

import (
	"fmt"
	"strings"
	"time"

	"github.com/getcarina/carina/common"
	libcarina "github.com/getcarina/libcarina"
	libmakeswarm "github.com/getcarina/libmakeswarm"
	"github.com/pkg/errors"
)

// MakeSwarm is an adapter between the cli and Carina (make-swarm)
type MakeSwarm struct {
	client  *libmakeswarm.ClusterClient
	Account *Account
}

// StatusNew is the status of a new, inactive cluster
const StatusNew = "new"

// StatusBuilding is the status of a cluster that is currently being built
const StatusBuilding = "building"

// StatusRebuilding is the status of a cluster that is currently rebuilding
const StatusRebuilding = "rebuilding-swarm"

const clusterPollingInterval = 10 * time.Second

func (carina *MakeSwarm) init() error {
	if carina.client == nil {
		carinaClient, err := carina.Account.Authenticate()
		if err != nil {
			return err
		}
		carina.client = carinaClient
	}
	return nil
}

// GetQuotas retrieves the quotas set for the account
func (carina *MakeSwarm) GetQuotas() (common.Quotas, error) {
	err := carina.init()
	if err != nil {
		return nil, err
	}

	common.Log.WriteDebug("[make-swarm] Retrieving account quotas")
	result, err := carina.client.GetQuotas()
	if err != nil {
		return nil, errors.Wrap(err, "[make-swarm] Unable to retrieve account quotas")
	}
	quotas := CarinaQuotas(*result)

	return &quotas, err
}

// CreateCluster creates a new cluster and prints the cluster information
func (carina *MakeSwarm) CreateCluster(name string, template string, nodes int) (common.Cluster, error) {
	err := carina.init()
	if err != nil {
		return nil, err
	}

	if template != "" {
		common.Log.WriteWarning("[make-swarm] Ignoring --template, not supported.")
	}

	common.Log.WriteDebug("[make-swarm] Creating %d-node cluster (%s)", nodes, name)
	options := libmakeswarm.Cluster{
		ClusterName: name,
		Nodes:       libmakeswarm.Number(nodes),
		AutoScale:   false, // Not exposing this since we are removing autoscale in make-coe
	}
	result, err := carina.client.Create(options)
	if err != nil {
		return nil, errors.Wrap(err, "[make-swarm] Unable to create the cluster")
	}
	cluster := &Cluster{Cluster: result}

	return cluster, err
}

// GetClusterCredentials retrieves the TLS certificates and configuration scripts for a cluster
func (carina *MakeSwarm) GetClusterCredentials(name string) (*libcarina.CredentialsBundle, error) {
	err := carina.init()
	if err != nil {
		return nil, err
	}

	common.Log.WriteDebug("[make-swarm] Retrieving cluster credentials (%s)", name)
	result, err := carina.client.GetCredentials(name)
	if err != nil {
		return nil, errors.Wrap(err, "[make-swarm] Unable to retrieve the cluster credentials")
	}

	creds := &libcarina.CredentialsBundle{Files: result.Files}

	return creds, nil
}

// ListClusters prints out a list of the user's clusters to the console
func (carina *MakeSwarm) ListClusters() ([]common.Cluster, error) {
	err := carina.init()
	if err != nil {
		return nil, err
	}

	common.Log.WriteDebug("[make-swarm] Listing clusters")
	results, err := carina.client.List()
	if err != nil {
		return nil, errors.Wrap(err, "[make-swarm] Unable to list clusters")
	}

	var clusters []common.Cluster
	for i := range results {
		cluster := &Cluster{Cluster: &results[i]}
		clusters = append(clusters, cluster)
	}

	return clusters, err
}

// ListClusterTemplates is not supported by make-swarm
func (carina *MakeSwarm) ListClusterTemplates() ([]common.ClusterTemplate, error) {
	return nil, errors.New("make-swarm does not support templates, use `carina create [cluster-name]` and omit the --template flag")
}

// RebuildCluster destroys and recreates the cluster
func (carina *MakeSwarm) RebuildCluster(name string) (common.Cluster, error) {
	err := carina.init()
	if err != nil {
		return nil, err
	}

	common.Log.WriteDebug("[make-swarm] Rebuilding cluster (%s)", name)
	result, err := carina.client.Rebuild(name)
	if err != nil {
		return nil, errors.Wrap(err, "[make-swarm] Unable to rebuild the cluster")
	}

	cluster := &Cluster{Cluster: result}

	return cluster, nil
}

// GetCluster prints out a cluster's information to the console
func (carina *MakeSwarm) GetCluster(name string) (common.Cluster, error) {
	err := carina.init()
	if err != nil {
		return nil, err
	}

	common.Log.WriteDebug("[make-swarm] Retrieving cluster (%s)", name)
	result, err := carina.client.Get(name)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("[make-swarm] Unable to retrieve cluster (%s)", name))
	}

	cluster := &Cluster{Cluster: result}

	return cluster, nil
}

// DeleteCluster permanently deletes a cluster
func (carina *MakeSwarm) DeleteCluster(name string) (common.Cluster, error) {
	err := carina.init()
	if err != nil {
		return nil, err
	}

	common.Log.WriteDebug("[make-swarm] Deleting cluster (%s)", name)
	result, err := carina.client.Delete(name)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("[make-swarm] Unable to delete cluster (%s)", name))
	}

	cluster := &Cluster{Cluster: result}

	return cluster, nil
}

// GrowCluster adds nodes to a cluster
func (carina *MakeSwarm) GrowCluster(name string, nodes int) (common.Cluster, error) {
	err := carina.init()
	if err != nil {
		return nil, err
	}

	common.Log.WriteDebug("[make-swarm] Growing cluster (%s) by %d nodes", name, nodes)
	result, err := carina.client.Grow(name, nodes)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("[make-swarm] Unable to grow cluster (%s)", name))
	}

	cluster := &Cluster{Cluster: result}

	return cluster, nil
}

// ResizeCluster resizes a cluster to the specified number of nodes by its id or name (if unique)
func (carina *MakeSwarm) ResizeCluster(token string, nodes int) (common.Cluster, error) {
	return nil, errors.New("[make-swarm] Resizing clusters from the carina cli is not supported. Please use 'grow'.")
}

// SetAutoScale enables or disables autoscaling on a cluster
func (carina *MakeSwarm) SetAutoScale(name string, value bool) (common.Cluster, error) {
	err := carina.init()
	if err != nil {
		return nil, err
	}

	common.Log.WriteDebug("[make-swarm] Changing the autoscale setting on the cluster (%s) to %t", name, value)
	result, err := carina.client.SetAutoScale(name, value)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("[make-swarm] Unable to change the cluster's autoscale setting (%s)", name))
	}

	cluster := &Cluster{Cluster: result}

	return cluster, nil
}

// WaitUntilClusterIsActive waits until the prior cluster operation is completed
func (carina *MakeSwarm) WaitUntilClusterIsActive(cluster common.Cluster) (common.Cluster, error) {
	isDone := func(cluster common.Cluster) bool {
		// Transitions past point of "new" or "building" are assumed to be active states
		status := strings.ToLower(cluster.GetStatus())
		return status != StatusNew && status != StatusBuilding && status != StatusRebuilding
	}

	if isDone(cluster) {
		return cluster, nil
	}

	for {
		cluster, err := carina.GetCluster(cluster.GetName())
		if err != nil {
			return cluster, err
		}

		if isDone(cluster) {
			return cluster, nil
		}

		common.Log.WriteDebug("[make-swarm] Waiting until cluster (%s) is active, currently in %s", cluster.GetName(), cluster.GetStatus())
		time.Sleep(clusterPollingInterval)
	}
}

// WaitUntilClusterIsDeleted returns the specified cluster, as make-swarm deletes immediately
func (carina *MakeSwarm) WaitUntilClusterIsDeleted(cluster common.Cluster) error {
	return nil
}
