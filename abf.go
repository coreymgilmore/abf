/*Package abf provides tooling to connect to the ABF Freight (Arc Best Freight) API. This
is for truck shipments.

You will need to have an ABF account and register for API access.

Currently this package can perform:
- pickup requests

Pickup API is a little odd.  You send the request as url parameters and get back an XML response.
The request can add up to 15 shipments in one request by incrementing a counter for some
parameters.

To create a pickup request:
- Get your ABF credentials.
- Create an item or items being shipped (Commodity{}).
- Create the pickup request adding the shipper and receiver details (PickupRequest{}).
- Add the item(s) to the pickup request.
- Request the pickup (RequestPickup()).
- Check for any errors.
*/
package abf

import (
	"bytes"
	"encoding/xml"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

//api urls
const (
	abfPickupURL = "https://www.abfs.com/xml/pickupxml.asp"
)

//testMode is set to "Y" by default and can be overridden by calling SetProductionMode()
//Setting this to "Y" will not schedule an actual pickup.
var testMode = "Y"

//timeout is the default time we should wait for a reply from ABF
//You may need to adjust this based on how slow connecting to ABF is for you.
//10 seconds is overly long, but sometimes ABF is very slow.
var timeout = time.Duration(10 * time.Second)

//types of requesters
const (
	RequesterShipper    = "1"
	RequesterConsignee  = "2"
	RequesterThirdParty = "3"
)

//type of payment terms
const (
	PayTermsPrepaid = "P"
	PayTermsCollect = "C"
)

//type of handling units
const (
	HandlingUnitPallet = "PLT"
)

//PickupRequest is the data sent to ABF to schedule a pickup
type PickupRequest struct {
	//required
	ID            string //api key
	RequesterType string //1 = shipper, 2 = consignee, 3 = third party
	PayTerms      string //"P" = prepaid, "C" = collect
	ShipContact   string //who to contact at the ship from location
	ShipName      string //name of company at ship from location
	ShipAddress   string
	ShipCity      string
	ShipState     string //two char code
	ShipZip       string
	ShipCountry   string //USA
	ShipPhone     string //xxxxxxxxxx
	ConsCity      string
	ConsState     string
	ConsZip       string
	ConsCountry   string      //two char code, US
	PickupDate    string      //mm/dd/yyyy
	AT            string      //time goods are available for pickup (hh:mm), 24 hour time
	OT            string      //open time of facility, should usually be the same at AT (hh:mm), 24 hour time
	CT            string      //close time of facility (hh:mm), 24 hour time
	Items         []Commodity //list of shipments to be picked up, up to 15

	//optional
	Test              string //Y or N
	RequesterName     string
	RequesterEmail    string
	RequesterPhone    string
	RequesterPhoneExt string
	ShipNamePlus      string //extra name info
	ShipPhoneExt      string
	ShipFax           string //xxxxxxxxxx
	ShipEmail         string
	ConsContact       string //who to contact at the receiver
	ConsName          string
	ConsNamePlus      string
	ConsAddress       string
	ConsPhone         string
	ConsPhoneExt      string
	ConsFax           string
	ConsEmail         string
	TPBContact        string
	TPBName           string
	TPBNamePlus       string
	TPBAddress        string
	TPBCity           string
	TPBState          string
	TPBZip            string
	TPBCountry        string
	TPBPhone          string
	TPBPhoneExt       string
	TPBFax            string
	TPBEmail          string
	Bol               string //bill of lading number (shipper reference)
	PO1               string //purchase order number (receiver reference)
	CRN1              string //other customer reference (invoice # maybe)
}

//Commodity is the data on a good being shipped
//up to 15 can be added per pickup request (**1-15)
//all are optional but obviously some must be given
type Commodity struct {
	HandlingUnits uint    //HN1, skid count
	UnitType      string  //HT1; PLT, no idea what valid values are
	Pieces        uint    //PN1
	PiecesType    string  //PT1, no idea what valid values are
	Weight        float64 //WT1, lbs
	Class         string  //CL1
	NMFC          string  //NMFC1
	NMFCSub       string  //SUB1
	Cube          float64 //CB1
	Description   string  //Desc1
	Hazmat        string  //"Y" or "N"
}

//Response is the data returned from ABF
//this is an xml
type Response struct {
	XMLName            xml.Name    `xml:"ABF"`
	ConfirmationNumber string      `xml:"CONFIRMATION"` //only returned when a pickup is successfully scheduled
	Ship               interface{} `xml:"SHIP"`         //shipper information
	Consignee          interface{} `xml:"CONS"`         //consignee info
	ThirdParty         interface{} `xml:"TPB"`          //third party info
	NumErrors          uint        `xml:"NUMERRORS"`    //0 if a confirmation number is returned
	Error              Error       `xml:"ERROR"`        //any error messages
}

//Error is any error from the request
type Error struct {
	Code    string `xml:"ERRORCODE"`
	Message string `xml:"ERRORMESSAGE"`
}

//SetProductionMode chooses the production url for use
func SetProductionMode(yes bool) {
	if yes {
		testMode = "N"
	}
	return
}

//SetTimeout updates the timeout value to something the user sets
//use this to increase the timeout if connecting to UPS is really slow
func SetTimeout(seconds time.Duration) {
	timeout = time.Duration(seconds * time.Second)
	return
}

//RequestPickup makes the api call to schedule the pickup
//this is a url with url parameters
func (p *PickupRequest) RequestPickup() (responseData Response, err error) {
	//set timeout
	httpClient := http.Client{
		Timeout: timeout,
	}

	//build the request parameters
	v := url.Values{}
	v.Add("ID", p.ID)
	v.Add("RequesterType", p.RequesterType)
	v.Add("PayTerms", p.PayTerms)
	v.Add("ShipContact", p.ShipContact)
	v.Add("ShipName", p.ShipName)
	v.Add("ShipAddress", p.ShipAddress)
	v.Add("ShipCity", p.ShipCity)
	v.Add("ShipState", p.ShipState)
	v.Add("ShipZip", p.ShipZip)
	v.Add("ShipCountry", p.ShipCountry)
	v.Add("ShipPhone", p.ShipPhone)
	v.Add("ConsCity", p.ConsCity)
	v.Add("ConsState", p.ConsState)
	v.Add("ConsZip", p.ConsZip)
	v.Add("ConsCountry", p.ConsCountry)
	v.Add("PickupDate", p.PickupDate)
	v.Add("AT", p.AT)
	v.Add("OT", p.OT)
	v.Add("CT", p.CT)
	v.Add("Bol", p.Bol)
	v.Add("PO1", p.PO1)
	v.Add("CRN1", p.CRN1)

	//set test mode
	v.Add("Test", testMode)

	log.Println(v.Encode())

	for index, item := range p.Items {
		v.Add("HN"+strconv.Itoa(index), strconv.Itoa(int(item.HandlingUnits)))
		v.Add("HT"+strconv.Itoa(index), item.UnitType)
		v.Add("PN"+strconv.Itoa(index), strconv.Itoa(int(item.Pieces)))
		v.Add("PT"+strconv.Itoa(index), item.PiecesType)
		v.Add("WT"+strconv.Itoa(index), strconv.FormatFloat(item.Weight, 'f', 0, 64))
	}

	res, err := httpClient.Post(abfPickupURL, "application/x-www-form-urlencoded", bytes.NewBufferString(v.Encode()))
	if err != nil {
		errors.Wrap(err, "abf.RequestPickup - could not make post request")
		return
	}

	//read the response
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		errors.Wrap(err, "abf.RequestPickup - could not read response")
		return
	}

	err = xml.Unmarshal(body, &responseData)
	if err != nil {
		errors.Wrap(err, "abf.RequestPickup - could not unmarshal response")
		return
	}

	//check if data was returned meaning request was successful
	//if not, reread the response data and log it
	if responseData.ConfirmationNumber == "" {
		log.Println("abf.RequestPickup - pickup request failed")
		log.Printf("%+v", responseData)

		//return our error so we know where this error came from, and UPS error message so we know what to fix
		err = errors.New("abf.RequestPickup - pickup request failed")
		err = errors.Wrap(err, responseData.Error.Message)
		return
	}

	//pickup request successful
	//response data will have confirmation number
	return
}
