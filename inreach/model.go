package inreach

import (
	"encoding/xml"
	"regexp"
)

type Fault struct {
	Code    int    `xml:"status_code"`
	Message string `xml:"msg"`
}

func (f *Fault) Error() string {
	return f.Message
}

const timeLayout = "1/2/2006 15:04:05 PM"

var (
	reElevation = regexp.MustCompile(`(?P<Elevation>[.0-9]+) m from MSL`)
)

const (
	// FieldID is the DeLorme internal ID for the event
	FieldID = "Id"
	// FieldTimeUTC is the US-formatted version of the event timestamp as UTC.
	FieldTimeUTC = "Time UTC"
	// FieldTime is the US-formatted version of the event timestamp in the preferred
	// timezone of the account owner.
	FieldTime = "Time"
	// FieldName is the First and last name of the user assigned to the device that
	// sent the message.
	FieldName = "Name"
	// FieldMapDisplayName is the Map Display Name for this user. This field is editable
	// by the user in their Account or Settings page.
	FieldMapDisplayName = "Map Display Name"
	// FieldDeviceType is the hardware type of the device in use. Recent types are:
	// DeLorme D1, InReach Zigbee, InReach Bluetooth, inReach SE, and inReach 2.5.
	// The D1 is known as model 1.0, the Zigbee and Bluetooth are model 1.5,
	// and most 2.5 devices are the Explorer model.
	FieldDeviceType = "Device Type"
	// FieldIMEI is the IMEI of the device sending the message.
	FieldIMEI = "IMEI"
	// FieldIncidentID is the emergency incident, if any.
	FieldIncidentID = "Incident Id"
	// FieldLatitude is the latitude in degrees WGS84, where negative is south of the equator.
	FieldLatitude = "Latitude"
	// FieldLongitude is the longitude in degrees WGS84, where negative is west of the Prime Meridian.
	FieldLongitude = "Longitude"
	// FieldElevation is the meters from Mean Sea Level.
	FieldElevation = "Elevation"
	// FieldVelocity is the ground speed of the device. Value is always in kilometers per hour.
	FieldVelocity = "Velocity"
	// FieldCourse is the approximate direction of travel of the device, always in true degrees.
	// Value is accurate to one of sixteen compass points only.
	FieldCourse = "Course"
	// FieldValidGPSFix is True if the device’s GPS unit has a fix. This is not a measure of
	// quality of the GPS fix. However, it is unlikely that any point will be provided without
	// a valid GPS fix.
	FieldValidGPS = "Valid GPS Fix"
	// FieldInEmergency is true if the device is in SOS state.
	FieldInEmergency = "In Emergency"
	// FieldText is the message text, if any, in Unicode.
	FieldText = "Text"
	// FieldEvent is the event log type. See `Events` table below.
	FieldEvent = "Event"
)

// Events IDs and descriptions from InReach
func Events() map[int]string {
	return map[int]string{
		01: "Emergency initiated from device",
		04: "Emergency confirmed",
		06: "Emergency canceled",
		13: "Text message received",
		16: "Location received",
		17: "Tracking message received",
		18: "Waypoint/navigation received",
		29: "Tracking turned off from device",
		30: "Tracking interval received",
		38: "Tracking turned on from device",
		41: "Reference point message received",
		43: "Text quick message received",
		45: "Message to shared map received",
		46: "Append MapShare to test message code received",
		52: "Quick Text to MapShare received",
		54: "Test message received",
		55: "Waypoint/navigation started",
		56: "Waypoint/navigation stopped",
	}
}

type LineString struct {
	Text        string `xml:",chardata"`
	Tessellate  string `xml:"tessellate"`
	Coordinates string `xml:"coordinates"`
}

type Point struct {
	Text         string `xml:",chardata"`
	Extrude      string `xml:"extrude"`
	AltitudeMode string `xml:"altitudeMode"`
	Coordinates  string `xml:"coordinates"`
}

type Data struct {
	Text  string `xml:",chardata"`
	Name  string `xml:"name,attr"`
	Value string `xml:"value"`
}

type ExtendedData struct {
	Text string  `xml:",chardata"`
	Data []*Data `xml:"Data"`
}

type TimeStamp struct {
	Text string `xml:",chardata"`
	When string `xml:"when"`
}

type Placemark struct {
	Text         string        `xml:",chardata"`
	Name         string        `xml:"name"`
	Visibility   string        `xml:"visibility"`
	Description  string        `xml:"description"`
	TimeStamp    *TimeStamp    `xml:"TimeStamp"`
	StyleURL     string        `xml:"styleUrl"`
	ExtendedData *ExtendedData `xml:"ExtendedData"`
	Point        *Point        `xml:"Point"`
	LineString   *LineString   `xml:"LineString"`
}

type Folder struct {
	Text      string       `xml:",chardata"`
	Name      string       `xml:"name"`
	Placemark []*Placemark `xml:"Placemark"`
}

type Document struct {
	Text   string  `xml:",chardata"`
	Name   string  `xml:"name"`
	Folder *Folder `xml:"Folder"`
}

type Feed struct {
	XMLName  xml.Name  `xml:"kml"`
	Text     string    `xml:",chardata"`
	XSD      string    `xml:"xsd,attr"`
	XSI      string    `xml:"xsi,attr"`
	XMLNS    string    `xml:"xmlns,attr"`
	Document *Document `xml:"Document"`
}
