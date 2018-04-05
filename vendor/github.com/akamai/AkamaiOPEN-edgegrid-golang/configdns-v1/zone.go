package dns

import (
	"fmt"
	"log"
	"sync"
	"time"

	"errors"
	"github.com/akamai/AkamaiOPEN-edgegrid-golang/client-v1"
)

type name struct {
	recordType string
	name       string
}

var (
	cnameNames    []name
	nonCnameNames []name
	zoneWriteLock sync.Mutex
)

// Zone represents a DNS zone
type Zone struct {
	Token string `json:"token"`
	Zone  struct {
		Name       string              `json:"name,omitempty"`
		A          []*ARecord          `json:"a,omitempty"`
		Aaaa       []*AaaaRecord       `json:"aaaa,omitempty"`
		Afsdb      []*AfsdbRecord      `json:"afsdb,omitempty"`
		Cname      []*CnameRecord      `json:"cname,omitempty"`
		Dnskey     []*DnskeyRecord     `json:"dnskey,omitempty"`
		Ds         []*DsRecord         `json:"ds,omitempty"`
		Hinfo      []*HinfoRecord      `json:"hinfo,omitempty"`
		Loc        []*LocRecord        `json:"loc,omitempty"`
		Mx         []*MxRecord         `json:"mx,omitempty"`
		Naptr      []*NaptrRecord      `json:"naptr,omitempty"`
		Ns         []*NsRecord         `json:"ns,omitempty"`
		Nsec3      []*Nsec3Record      `json:"nsec3,omitempty"`
		Nsec3param []*Nsec3paramRecord `json:"nsec3param,omitempty"`
		Ptr        []*PtrRecord        `json:"ptr,omitempty"`
		Rp         []*RpRecord         `json:"rp,omitempty"`
		Rrsig      []*RrsigRecord      `json:"rrsig,omitempty"`
		Soa        *SoaRecord          `json:"soa,omitempty"`
		Spf        []*SpfRecord        `json:"spf,omitempty"`
		Srv        []*SrvRecord        `json:"srv,omitempty"`
		Sshfp      []*SshfpRecord      `json:"sshfp,omitempty"`
		Txt        []*TxtRecord        `json:"txt,omitempty"`
	} `json:"zone"`
}

// NewZone creates a new Zone
func NewZone(hostname string) *Zone {
	zone := &Zone{Token: "new"}
	zone.Zone.Soa = NewSoaRecord()
	zone.Zone.Name = hostname
	return zone
}

// GetZone retrieves a DNS Zone for a given hostname
func GetZone(hostname string) (*Zone, error) {
	zone := NewZone(hostname)
	req, err := client.NewRequest(
		Config,
		"GET",
		"/config-dns/v1/zones/"+hostname,
		nil,
	)
	if err != nil {
		return nil, err
	}

	res, err := client.Do(Config, req)
	if err != nil {
		return nil, err
	}

	if client.IsError(res) && res.StatusCode != 404 {
		return nil, client.NewAPIError(res)
	} else if res.StatusCode == 404 {
		return nil, &ZoneError{zoneName: hostname}
	} else {
		err = client.BodyJSON(res, &zone)
		if err != nil {
			return nil, err
		}

		return zone, nil
	}
}

// Save updates the Zone
func (zone *Zone) Save() error {
	// This lock will restrict the concurrency of API calls
	// to 1 save request at a time. This is needed for the Soa.Serial value which
	// is required to be incremented for every subsequent update to a zone
	// so we have to save just one request at a time to ensure this is always
	// incremented properly
	zoneWriteLock.Lock()
	defer zoneWriteLock.Unlock()

	valid, f := zone.validateCnames()
	if valid == false {
		var msg string
		for _, v := range f {
			msg = msg + fmt.Sprintf("\n%s Record '%s' conflicts with CNAME", v.recordType, v.name)
		}
		return &ZoneError{
			zoneName:        zone.Zone.Name,
			apiErrorMessage: "All CNAMEs must be unique in the zone" + msg,
		}
	}

	req, err := client.NewJSONRequest(
		Config,
		"POST",
		"/config-dns/v1/zones/"+zone.Zone.Name,
		zone,
	)
	if err != nil {
		return err
	}

	res, err := client.Do(Config, req)

	// Network error
	if err != nil {
		return &ZoneError{
			zoneName:         zone.Zone.Name,
			httpErrorMessage: err.Error(),
			err:              err,
		}
	}

	// API error
	if client.IsError(res) {
		err := client.NewAPIError(res)
		return &ZoneError{zoneName: zone.Zone.Name, apiErrorMessage: err.Title, err: err}
	}

	for {
		updatedZone, err := GetZone(zone.Zone.Name)
		if err != nil {
			return err
		}

		if updatedZone.Token != zone.Token {
			log.Printf("[TRACE] Token updated: old: %s, new: %s", zone.Token, updatedZone.Token)
			*zone = *updatedZone
			break
		}
		log.Println("[DEBUG] Token not updated, retrying...")
		time.Sleep(time.Second)
	}

	log.Printf("[INFO] Zone Saved")

	return nil
}

func (zone *Zone) Delete() error {
	// remove all the records except for SOA
	// which is required and save the zone
	zone.Zone.A = nil
	zone.Zone.Aaaa = nil
	zone.Zone.Afsdb = nil
	zone.Zone.Cname = nil
	zone.Zone.Dnskey = nil
	zone.Zone.Ds = nil
	zone.Zone.Hinfo = nil
	zone.Zone.Loc = nil
	zone.Zone.Mx = nil
	zone.Zone.Naptr = nil
	zone.Zone.Ns = nil
	zone.Zone.Nsec3 = nil
	zone.Zone.Nsec3param = nil
	zone.Zone.Ptr = nil
	zone.Zone.Rp = nil
	zone.Zone.Rrsig = nil
	zone.Zone.Spf = nil
	zone.Zone.Srv = nil
	zone.Zone.Sshfp = nil
	zone.Zone.Txt = nil

	return zone.Save()
}

func (zone *Zone) AddRecord(recordPtr interface{}) error {
	switch recordPtr.(type) {
	case *ARecord:
		zone.addARecord(recordPtr.(*ARecord))
	case *AaaaRecord:
		zone.addAaaaRecord(recordPtr.(*AaaaRecord))
	case *AfsdbRecord:
		zone.addAfsdbRecord(recordPtr.(*AfsdbRecord))
	case *CnameRecord:
		zone.addCnameRecord(recordPtr.(*CnameRecord))
	case *DnskeyRecord:
		zone.addDnskeyRecord(recordPtr.(*DnskeyRecord))
	case *DsRecord:
		zone.addDsRecord(recordPtr.(*DsRecord))
	case *HinfoRecord:
		zone.addHinfoRecord(recordPtr.(*HinfoRecord))
	case *LocRecord:
		zone.addLocRecord(recordPtr.(*LocRecord))
	case *MxRecord:
		zone.addMxRecord(recordPtr.(*MxRecord))
	case *NaptrRecord:
		zone.addNaptrRecord(recordPtr.(*NaptrRecord))
	case *NsRecord:
		zone.addNsRecord(recordPtr.(*NsRecord))
	case *Nsec3Record:
		zone.addNsec3Record(recordPtr.(*Nsec3Record))
	case *Nsec3paramRecord:
		zone.addNsec3paramRecord(recordPtr.(*Nsec3paramRecord))
	case *PtrRecord:
		zone.addPtrRecord(recordPtr.(*PtrRecord))
	case *RpRecord:
		zone.addRpRecord(recordPtr.(*RpRecord))
	case *RrsigRecord:
		zone.addRrsigRecord(recordPtr.(*RrsigRecord))
	case *SoaRecord:
		zone.addSoaRecord(recordPtr.(*SoaRecord))
	case *SpfRecord:
		zone.addSpfRecord(recordPtr.(*SpfRecord))
	case *SrvRecord:
		zone.addSrvRecord(recordPtr.(*SrvRecord))
	case *SshfpRecord:
		zone.addSshfpRecord(recordPtr.(*SshfpRecord))
	case *TxtRecord:
		zone.addTxtRecord(recordPtr.(*TxtRecord))
	}

	return nil
}

func (zone *Zone) RemoveRecord(recordPtr interface{}) error {
	switch recordPtr.(type) {
	case *ARecord:
		return zone.removeARecord(recordPtr.(*ARecord))
	case *AaaaRecord:
		return zone.removeAaaaRecord(recordPtr.(*AaaaRecord))
	case *AfsdbRecord:
		return zone.removeAfsdbRecord(recordPtr.(*AfsdbRecord))
	case *CnameRecord:
		return zone.removeCnameRecord(recordPtr.(*CnameRecord))
	case *DnskeyRecord:
		return zone.removeDnskeyRecord(recordPtr.(*DnskeyRecord))
	case *DsRecord:
		return zone.removeDsRecord(recordPtr.(*DsRecord))
	case *HinfoRecord:
		return zone.removeHinfoRecord(recordPtr.(*HinfoRecord))
	case *LocRecord:
		return zone.removeLocRecord(recordPtr.(*LocRecord))
	case *MxRecord:
		return zone.removeMxRecord(recordPtr.(*MxRecord))
	case *NaptrRecord:
		return zone.removeNaptrRecord(recordPtr.(*NaptrRecord))
	case *NsRecord:
		return zone.removeNsRecord(recordPtr.(*NsRecord))
	case *Nsec3Record:
		return zone.removeNsec3Record(recordPtr.(*Nsec3Record))
	case *Nsec3paramRecord:
		return zone.removeNsec3paramRecord(recordPtr.(*Nsec3paramRecord))
	case *PtrRecord:
		return zone.removePtrRecord(recordPtr.(*PtrRecord))
	case *RpRecord:
		return zone.removeRpRecord(recordPtr.(*RpRecord))
	case *RrsigRecord:
		return zone.removeRrsigRecord(recordPtr.(*RrsigRecord))
	case *SoaRecord:
		return zone.removeSoaRecord(recordPtr.(*SoaRecord))
	case *SpfRecord:
		return zone.removeSpfRecord(recordPtr.(*SpfRecord))
	case *SrvRecord:
		return zone.removeSrvRecord(recordPtr.(*SrvRecord))
	case *SshfpRecord:
		return zone.removeSshfpRecord(recordPtr.(*SshfpRecord))
	case *TxtRecord:
		return zone.removeTxtRecord(recordPtr.(*TxtRecord))
	}

	return nil
}

func (zone *Zone) addARecord(record *ARecord) {
	zone.Zone.A = append(zone.Zone.A, record)
	nonCnameNames = append(nonCnameNames, name{recordType: "A", name: record.Name})
}

func (zone *Zone) addAaaaRecord(record *AaaaRecord) {
	zone.Zone.Aaaa = append(zone.Zone.Aaaa, record)
	nonCnameNames = append(nonCnameNames, name{recordType: "AAAA", name: record.Name})
}

func (zone *Zone) addAfsdbRecord(record *AfsdbRecord) {
	zone.Zone.Afsdb = append(zone.Zone.Afsdb, record)
	nonCnameNames = append(nonCnameNames, name{recordType: "AFSDB", name: record.Name})
}

func (zone *Zone) addCnameRecord(record *CnameRecord) {
	zone.Zone.Cname = append(zone.Zone.Cname, record)
	cnameNames = append(cnameNames, name{recordType: "CNAME", name: record.Name})
}

func (zone *Zone) addDnskeyRecord(record *DnskeyRecord) {
	zone.Zone.Dnskey = append(zone.Zone.Dnskey, record)
	nonCnameNames = append(nonCnameNames, name{recordType: "DNSKEY", name: record.Name})
}

func (zone *Zone) addDsRecord(record *DsRecord) {
	zone.Zone.Ds = append(zone.Zone.Ds, record)
	nonCnameNames = append(nonCnameNames, name{recordType: "DS", name: record.Name})
}

func (zone *Zone) addHinfoRecord(record *HinfoRecord) {
	zone.Zone.Hinfo = append(zone.Zone.Hinfo, record)
	nonCnameNames = append(nonCnameNames, name{recordType: "HINFO", name: record.Name})
}

func (zone *Zone) addLocRecord(record *LocRecord) {
	zone.Zone.Loc = append(zone.Zone.Loc, record)
	nonCnameNames = append(nonCnameNames, name{recordType: "LOC", name: record.Name})
}

func (zone *Zone) addMxRecord(record *MxRecord) {
	zone.Zone.Mx = append(zone.Zone.Mx, record)
	nonCnameNames = append(nonCnameNames, name{recordType: "MX", name: record.Name})
}

func (zone *Zone) addNaptrRecord(record *NaptrRecord) {
	zone.Zone.Naptr = append(zone.Zone.Naptr, record)
	nonCnameNames = append(nonCnameNames, name{recordType: "NAPTR", name: record.Name})
}

func (zone *Zone) addNsRecord(record *NsRecord) {
	zone.Zone.Ns = append(zone.Zone.Ns, record)
	nonCnameNames = append(nonCnameNames, name{recordType: "NS", name: record.Name})
}

func (zone *Zone) addNsec3Record(record *Nsec3Record) {
	zone.Zone.Nsec3 = append(zone.Zone.Nsec3, record)
	nonCnameNames = append(nonCnameNames, name{recordType: "NSEC3", name: record.Name})
}

func (zone *Zone) addNsec3paramRecord(record *Nsec3paramRecord) {
	zone.Zone.Nsec3param = append(zone.Zone.Nsec3param, record)
	nonCnameNames = append(nonCnameNames, name{recordType: "NSEC3PARAM", name: record.Name})
}

func (zone *Zone) addPtrRecord(record *PtrRecord) {
	zone.Zone.Ptr = append(zone.Zone.Ptr, record)
	nonCnameNames = append(nonCnameNames, name{recordType: "PTR", name: record.Name})
}

func (zone *Zone) addRpRecord(record *RpRecord) {
	zone.Zone.Rp = append(zone.Zone.Rp, record)
	nonCnameNames = append(nonCnameNames, name{recordType: "RP", name: record.Name})
}

func (zone *Zone) addRrsigRecord(record *RrsigRecord) {
	zone.Zone.Rrsig = append(zone.Zone.Rrsig, record)
	nonCnameNames = append(nonCnameNames, name{recordType: "RRSIG", name: record.Name})
}

func (zone *Zone) addSoaRecord(record *SoaRecord) {
	// Only one SOA records is allowed
	zone.Zone.Soa = record
}

func (zone *Zone) addSpfRecord(record *SpfRecord) {
	zone.Zone.Spf = append(zone.Zone.Spf, record)
	nonCnameNames = append(nonCnameNames, name{recordType: "SPF", name: record.Name})
}

func (zone *Zone) addSrvRecord(record *SrvRecord) {
	zone.Zone.Srv = append(zone.Zone.Srv, record)
	nonCnameNames = append(nonCnameNames, name{recordType: "SRV", name: record.Name})
}

func (zone *Zone) addSshfpRecord(record *SshfpRecord) {
	zone.Zone.Sshfp = append(zone.Zone.Sshfp, record)
	nonCnameNames = append(nonCnameNames, name{recordType: "SSHFP", name: record.Name})
}

func (zone *Zone) addTxtRecord(record *TxtRecord) {
	zone.Zone.Txt = append(zone.Zone.Txt, record)
	nonCnameNames = append(nonCnameNames, name{recordType: "TXT", name: record.Name})
}

func (zone *Zone) removeARecord(record *ARecord) error {
	var found bool
	for key, r := range zone.Zone.A {
		if r == record {
			records := zone.Zone.A[:key]
			zone.Zone.A = append(records, zone.Zone.A[key+1:]...)
			found = true
		}
	}

	if !found {
		return errors.New("A Record not found")
	}

	zone.removeNonCnameName(record.Name)

	return nil
}

func (zone *Zone) removeAaaaRecord(record *AaaaRecord) error {
	var found bool
	for key, r := range zone.Zone.Aaaa {
		if r == record {
			records := zone.Zone.Aaaa[:key]
			zone.Zone.Aaaa = append(records, zone.Zone.Aaaa[key+1:]...)
			found = true
		}
	}

	if !found {
		return errors.New("AAAA Record not found")
	}

	zone.removeNonCnameName(record.Name)

	return nil
}

func (zone *Zone) removeAfsdbRecord(record *AfsdbRecord) error {
	var found bool
	for key, r := range zone.Zone.Afsdb {
		if r == record {
			records := zone.Zone.Afsdb[:key]
			zone.Zone.Afsdb = append(records, zone.Zone.Afsdb[key+1:]...)
			found = true
		}
	}

	if !found {
		return errors.New("Afsdb Record not found")
	}

	zone.removeNonCnameName(record.Name)

	return nil
}

func (zone *Zone) removeCnameRecord(record *CnameRecord) error {
	var found bool
	for key, r := range zone.Zone.Cname {
		if r == record {
			records := zone.Zone.Cname[:key]
			zone.Zone.Cname = append(records, zone.Zone.Cname[key+1:]...)
			found = true
		}
	}

	if !found {
		return errors.New("Cname Record not found")
	}

	zone.removeCnameName(record.Name)

	return nil
}

func (zone *Zone) removeDnskeyRecord(record *DnskeyRecord) error {
	var found bool
	for key, r := range zone.Zone.Dnskey {
		if r == record {
			records := zone.Zone.Dnskey[:key]
			zone.Zone.Dnskey = append(records, zone.Zone.Dnskey[key+1:]...)
			found = true
		}
	}

	if !found {
		return errors.New("Dnskey Record not found")
	}

	zone.removeNonCnameName(record.Name)

	return nil
}

func (zone *Zone) removeDsRecord(record *DsRecord) error {
	var found bool
	for key, r := range zone.Zone.Ds {
		if r == record {
			records := zone.Zone.Ds[:key]
			zone.Zone.Ds = append(records, zone.Zone.Ds[key+1:]...)
			found = true
		}
	}

	if !found {
		return errors.New("Ds Record not found")
	}

	zone.removeNonCnameName(record.Name)

	return nil
}

func (zone *Zone) removeHinfoRecord(record *HinfoRecord) error {
	var found bool
	for key, r := range zone.Zone.Hinfo {
		if r == record {
			records := zone.Zone.Hinfo[:key]
			zone.Zone.Hinfo = append(records, zone.Zone.Hinfo[key+1:]...)
			found = true
		}
	}

	if !found {
		return errors.New("Hinfo Record not found")
	}

	zone.removeNonCnameName(record.Name)

	return nil
}

func (zone *Zone) removeLocRecord(record *LocRecord) error {
	var found bool
	for key, r := range zone.Zone.Loc {
		if r == record {
			records := zone.Zone.Loc[:key]
			zone.Zone.Loc = append(records, zone.Zone.Loc[key+1:]...)
			found = true
		}
	}

	if !found {
		return errors.New("Loc Record not found")
	}

	zone.removeNonCnameName(record.Name)

	return nil
}

func (zone *Zone) removeMxRecord(record *MxRecord) error {
	var found bool
	for key, r := range zone.Zone.Mx {
		if r == record {
			records := zone.Zone.Mx[:key]
			zone.Zone.Mx = append(records, zone.Zone.Mx[key+1:]...)
			found = true
		}
	}

	if !found {
		return errors.New("Mx Record not found")
	}

	zone.removeNonCnameName(record.Name)

	return nil
}

func (zone *Zone) removeNaptrRecord(record *NaptrRecord) error {
	var found bool
	for key, r := range zone.Zone.Naptr {
		if r == record {
			records := zone.Zone.Naptr[:key]
			zone.Zone.Naptr = append(records, zone.Zone.Naptr[key+1:]...)
			found = true
		}
	}

	if !found {
		return errors.New("Naptr Record not found")
	}

	zone.removeNonCnameName(record.Name)

	return nil
}

func (zone *Zone) removeNsRecord(record *NsRecord) error {
	var found bool
	for key, r := range zone.Zone.Ns {
		if r == record {
			records := zone.Zone.Ns[:key]
			zone.Zone.Ns = append(records, zone.Zone.Ns[key+1:]...)
			found = true
		}
	}

	if !found {
		return errors.New("Ns Record not found")
	}

	zone.removeNonCnameName(record.Name)

	return nil
}

func (zone *Zone) removeNsec3Record(record *Nsec3Record) error {
	var found bool
	for key, r := range zone.Zone.Nsec3 {
		if r == record {
			records := zone.Zone.Nsec3[:key]
			zone.Zone.Nsec3 = append(records, zone.Zone.Nsec3[key+1:]...)
			found = true
		}
	}

	if !found {
		return errors.New("Nsec3 Record not found")
	}

	zone.removeNonCnameName(record.Name)

	return nil
}

func (zone *Zone) removeNsec3paramRecord(record *Nsec3paramRecord) error {
	var found bool
	for key, r := range zone.Zone.Nsec3param {
		if r == record {
			records := zone.Zone.Nsec3param[:key]
			zone.Zone.Nsec3param = append(records, zone.Zone.Nsec3param[key+1:]...)
			found = true
		}
	}

	if !found {
		return errors.New("Nsec3param Record not found")
	}

	zone.removeNonCnameName(record.Name)

	return nil
}

func (zone *Zone) removePtrRecord(record *PtrRecord) error {
	var found bool
	for key, r := range zone.Zone.Ptr {
		if r == record {
			records := zone.Zone.Ptr[:key]
			zone.Zone.Ptr = append(records, zone.Zone.Ptr[key+1:]...)
			found = true
		}
	}

	if !found {
		return errors.New("Ptr Record not found")
	}

	zone.removeNonCnameName(record.Name)

	return nil
}

func (zone *Zone) removeRpRecord(record *RpRecord) error {
	var found bool
	for key, r := range zone.Zone.Rp {
		if r == record {
			records := zone.Zone.Rp[:key]
			zone.Zone.Rp = append(records, zone.Zone.Rp[key+1:]...)
			found = true
		}
	}

	if !found {
		return errors.New("Rp Record not found")
	}

	zone.removeNonCnameName(record.Name)

	return nil
}

func (zone *Zone) removeRrsigRecord(record *RrsigRecord) error {
	var found bool
	for key, r := range zone.Zone.Rrsig {
		if r == record {
			records := zone.Zone.Rrsig[:key]
			zone.Zone.Rrsig = append(records, zone.Zone.Rrsig[key+1:]...)
			found = true
		}
	}

	if !found {
		return errors.New("Rrsig Record not found")
	}

	zone.removeNonCnameName(record.Name)

	return nil
}

func (zone *Zone) removeSoaRecord(record *SoaRecord) error {
	zone.Zone.Soa = record
	return nil
}

func (zone *Zone) removeSpfRecord(record *SpfRecord) error {
	var found bool
	for key, r := range zone.Zone.Spf {
		if r == record {
			records := zone.Zone.Spf[:key]
			zone.Zone.Spf = append(records, zone.Zone.Spf[key+1:]...)
			found = true
		}
	}

	if !found {
		return errors.New("Spf Record not found")
	}

	zone.removeNonCnameName(record.Name)

	return nil
}

func (zone *Zone) removeSrvRecord(record *SrvRecord) error {
	var found bool
	for key, r := range zone.Zone.Srv {
		if r == record {
			records := zone.Zone.Srv[:key]
			zone.Zone.Srv = append(records, zone.Zone.Srv[key+1:]...)
			found = true
		}
	}

	if !found {
		return errors.New("Srv Record not found")
	}

	zone.removeNonCnameName(record.Name)

	return nil
}

func (zone *Zone) removeSshfpRecord(record *SshfpRecord) error {
	var found bool
	for key, r := range zone.Zone.Sshfp {
		if r == record {
			records := zone.Zone.Sshfp[:key]
			zone.Zone.Sshfp = append(records, zone.Zone.Sshfp[key+1:]...)
			found = true
		}
	}

	if !found {
		return errors.New("Sshfp Record not found")
	}

	zone.removeNonCnameName(record.Name)

	return nil
}

func (zone *Zone) removeTxtRecord(record *TxtRecord) error {
	var found bool
	for key, r := range zone.Zone.Txt {
		if r == record {
			records := zone.Zone.Txt[:key]
			zone.Zone.Txt = append(records, zone.Zone.Txt[key+1:]...)
			found = true
		}
	}

	if !found {
		return errors.New("Txt Record not found")
	}

	zone.removeNonCnameName(record.Name)

	return nil
}

func (zone *Zone) PreMarshalJSON() error {
	if zone.Zone.Soa.Serial > 0 {
		zone.Zone.Soa.Serial = zone.Zone.Soa.Serial + 1
	} else {
		zone.Zone.Soa.Serial = int(time.Now().Unix())
	}
	return nil
}

func (zone *Zone) validateCnames() (bool, []name) {
	var valid bool = true
	var failedRecords []name
	for _, v := range cnameNames {
		for _, vv := range nonCnameNames {
			if v.name == vv.name {
				valid = false
				failedRecords = append(failedRecords, vv)
			}
		}
	}
	return valid, failedRecords
}

func (zone *Zone) removeCnameName(host string) {
	for i, v := range cnameNames {
		if v.name == host {
			r := cnameNames[:i]
			cnameNames = append(r, cnameNames[i+1:]...)
		}
	}
}

func (zone *Zone) removeNonCnameName(host string) {
	for i, v := range nonCnameNames {
		if v.name == host {
			r := nonCnameNames[:i]
			nonCnameNames = append(r, nonCnameNames[i+1:]...)
		}
	}
}
