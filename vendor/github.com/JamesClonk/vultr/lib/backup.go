package lib

import (
	"fmt"
	"net/url"
	"sort"
	"time"
)

// Backup of a virtual machine
type Backup struct {
	ID          string `json:"BACKUPID"`
	Created     string `json:"date_created"`
	Description string `json:"description"`
	Size        string `json:"size"`
	Status      string `json:"status"`
}

type backups []Backup

func (bs backups) Len() int      { return len(bs) }
func (bs backups) Swap(i, j int) { bs[i], bs[j] = bs[j], bs[i] }

// sort by most recent
func (bs backups) Less(i, j int) bool {
	timeLayout := "2006-01-02 15:04:05" // oh my : https://golang.org/src/time/format.go
	t1, _ := time.Parse(timeLayout, bs[i].Created)
	t2, _ := time.Parse(timeLayout, bs[j].Created)
	return t1.After(t2)
}

// GetBackups retrieves a list of all backups on Vultr account
func (c *Client) GetBackups(id string, backupid string) ([]Backup, error) {
	var backupMap map[string]Backup
	values := url.Values{
		"SUBID":    {id},
		"BACKUPID": {backupid},
	}

	if err := c.post(`backup/list`, values, &backupMap); err != nil {
		return nil, err
	}

	var backup []Backup
	for _, b := range backupMap {
		fmt.Println(b)
		backup = append(backup, b)
	}
	sort.Sort(backups(backup))
	return backup, nil
}
