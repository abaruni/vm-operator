package vsphere

import (
	"context"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
)

type Datacenter struct {
	client     govmomi.Client
	name       string
	Datacenter *object.Datacenter
	finder     *find.Finder
}

func NewDatacenter(client govmomi.Client, name string) (*Datacenter, error) {
	return &Datacenter{client: client, name: name}, nil
}

func (dc *Datacenter) Lookup() error {

	if dc.finder == nil {
		dc.finder = find.NewFinder(dc.client.Client, false)
	}

	// Datacenter is not required (ls command for example).
	// Set for relative func if dc flag is given or
	// if there is a single (default) Datacenter
	ctx := context.TODO()
	var err error = nil
	if dc.Datacenter, err = dc.finder.Datacenter(ctx, dc.name); err != nil {
		return err
	}

	dc.finder.SetDatacenter(dc.Datacenter)

	return nil
}