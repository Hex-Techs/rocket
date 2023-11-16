package clusters

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hex-techs/rocket/pkg/models/cluster"
	"github.com/hex-techs/rocket/pkg/rocketctl"
	"github.com/hex-techs/rocket/pkg/utils/display"
	"github.com/hex-techs/rocket/pkg/utils/web"
	"github.com/jedib0t/go-pretty/v6/table"
)

func List(filename string) error {
	cli, err := rocketctl.InitClient(filename)
	if err != nil {
		return err
	}
	var clusterItems []cluster.Cluster
	if err := cli.List(&web.Request{Page: 1, Limit: -1}, &clusterItems); err != nil {
		return err
	}
	d := display.NewTerminalDisplay("", 0)
	header := table.Row{"name", "region", "cloud", "mode", "create", "state"}
	var content [][]interface{}
	for _, c := range clusterItems {
		mode := "hub"
		if c.Agent {
			mode = "edge"
		}
		content = append(content, []interface{}{c.Name, c.Region, c.CloudArea, mode, c.CreatedAt.Local().Format(time.DateTime), c.State})
	}
	d.Table(header, content...)
	return nil
}

func Get(filename, name string) error {
	cli, err := rocketctl.InitClient(filename)
	if err != nil {
		return err
	}
	var cls cluster.Cluster
	if err := cli.Get(name, nil, &cls); err != nil {
		return err
	}
	b, err := json.MarshalIndent(cls, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

func Approved(filename string, name string) error {
	cli, err := rocketctl.InitClient(filename)
	if err != nil {
		return err
	}
	var clusterItem cluster.Cluster
	if err := cli.Get(name, nil, &clusterItem); err != nil {
		return err
	}
	if clusterItem.State == cluster.Approve {
		return fmt.Errorf("the cluster '%s' has been in state of approved", name)
	}
	d := display.NewTerminalDisplay(fmt.Sprintf("The cluster '%s' is in the state of '%s', Are you sure to approve?",
		clusterItem.Name, clusterItem.State), 0)
	if d.Confirm() {
		clusterItem.State = cluster.Approve
		return cli.Update(name, &clusterItem)
	}
	return nil
}

func Create(filename string, clusterItem *cluster.Cluster) error {
	cli, err := rocketctl.InitClient(filename)
	if err != nil {
		return err
	}
	return cli.Create(clusterItem)
}
