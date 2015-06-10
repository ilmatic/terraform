package google

import (
	"fmt"
	"log"
	"time"
	"errors"

	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/container/v1beta1"
	"google.golang.org/api/googleapi"
)

func resourceContainerCluster() *schema.Resource {
	return &schema.Resource{
		Create: resourceContainerClusterCreate,
		Read: resourceContainerClusterRead,
		Delete: resourceContainerClusterDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type: schema.TypeString,
				Required: true,
			},
			"master": &schema.Schema{
				Type: schema.TypeString,
				Computed: true,
			},
			"workers": &schema.Schema{
				Type: schema.TypeList,
				Computed: true,
			},
		},
	}
}

func resourceContainerClusterCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	cluster := &container.Cluster{
		Name: d.Get("name").(string),
	}

	clusterRequest := &container.CreateClusterRequest{
		Cluster: cluster,
	}

	log.Printf("[DEBUG] Cluster insert request: %#v", cluster)

	op, err := config.clientContainer.Projects.Zones.Clusters.Create(config.Project, d.Get("zone").(string), clusterRequest).Do()

	if err != nil {
		return fmt.Errorf("Error creating cluster: %s", err)
	}

	// It probably maybe worked, so store the ID now
	d.SetId(cluster.Name)

	// Wait for the operation to complete
	w := &ContainerOperationWaiter{
		Service: config.clientContainer,
		Op: op,
		Project: config.Project,
		Region: config.Region,
		Type: OperationWaitRegion,
	}

	state := w.Conf()
	state.Timeout = 2 * time.Minute
	state.MinTimeout = 1 * time.Second

	opRaw, err := state.WaitForState()

	if err != nil {
		return fmt.Errorf("Error waiting for cluster to create: %s", err)
	}

	op = opRaw.(*container.Operation)

	if op.ErrorMessage != "" {
		// The resource didn't actually create
		d.SetId("")

		// Return the error
		return errors.New(op.ErrorMessage)
	}

	return resourceContainerClusterRead(d, meta)
}

func resourceContainerClusterRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	cluster, err := config.clientContainer.Projects.Zones.Clusters.Get(config.Project, d.Get("zone").(string), d.Id()).Do()

	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			// The resource doesn't exist anymore
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error reading cluster: %s", err)
	}

	d.Set("self_link", cluster.SelfLink)

	return nil
}

func resourceContainerClusterDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Delete the cluster
	log.Printf("[DEBUG] cluster delete request")

	op, err := config.clientContainer.Projects.Zones.Clusters.Delete(config.Project, d.Get("zone").(string), d.Id()).Do()

	if err != nil {
		return fmt.Errorf("Error deleting cluster: %s", err)
	}

	// Wait for the operation to complete
	w := &ContainerOperationWaiter{
		Service: config.clientContainer,
		Op: op,
		Project: config.Project,
		Region: config.Region,
		Type: OperationWaitZone,
	}

	state := w.Conf()
	state.Timeout = 2 * time.Minute
	state.MinTimeout = 1 * time.Second

	opRaw, err := state.WaitForState()

	if err != nil {
		return fmt.Errorf("Error waiting for cluster to delete: %s", err)
	}

	op = opRaw.(*container.Operation)
	if op.ErrorMessage != "" {
		// Return the error
		return errors.New(op.ErrorMessage)
	}

	d.SetId("")
	return nil
}
