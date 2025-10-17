/*

Device Requests
bmRequestType	bRequest	Description
1000 0000b	GET_STATUS
( 0 )	Returns the status of the device. Primarily used to determine if the device is capable of Remote Wake-up, and whether or not the device is Self or Bus-powered.
0000 0000b	CLEAR_FEATURE
(01)	Disables either the DEVICE_REMOTE_WAKEUP or the TEST_MODE feature.
0000 0000b	SET_FEATURE
(03)	Enables either the DEVICE_REMOTE_WAKEUP or the TEST_MODE feature.
0000 0000b	SET_ADDRESS
(05)	During enumeration this instruction is used to assign an address (1 -127) to the device.
1000 0000b	GET_DESCRIPTOR
(06)	Returns the descriptor table selected by the wValue parameter.
0000 0000b	SET_DESCRIPTOR
(07)	Sets the specified descriptor value.
1000 0000b	GET_CONFIGURATION
(08)	Returns the index value for the active device configuration.
0000 0000b	SET_CONFIGURATION
(09)	Cause the specified device configuration to become active.

Interface Requests
bmRequestType	bRequest	Description
1000 0001b	GET_STATUS
(0)	Returns the status of the interface. Currently both returned bytes are reserved for future use.
0000 0001b	CLEAR_FEATURE
(01)	Disables an interface feature.
0000 0001b	SET_FEATURE
(03)	Enables the specified interface feature.
1000 0001b	GET_INTERFACE
(0A)	Retrieve the index for the currently active interface.
0000 0001b	SET_INTERFACE
(11)	Indicated with interface to activate.


Endpoint Requests
bmRequestType	bRequest	Description
1000 0010b	GET_STATUS
(0)	Returns the status of the endpoint.
0000 0010b	CLEAR_FEATURE
(01)	Disables an endpoint feature.
0000 0010b	SET_FEATURE
(03)	Enable endpoint feature.
1000 0010b	SYNCH_FRAME
(12)	Used to report endpoint synchronization frame.

Hub Requests
Get Hub Status (GET_STATUS)
Get Port Status (GET_STATUS)
Clear Hub Feature (CLEAR_FEATURE)
Clear Port Feature (CLEAR_FEATURE)
Get Bus State (GET_STATE) obsolete since USB 2.0
Set Hub Feature (SET_FEATURE)
Set Port Feature (SET_FEATURE)
Get Hub Descriptor (GET_DESCRIPTOR)
Set Hub Descriptor (SET_DESCRIPTOR)
Clear TT Buffer (CLEAR_TT_BUFFER)
Reset TT (RESET_TT)
Get TT State (GET_TT_STATE)
Stop TT (STOP_TT)

*/

/*

Each control request starts with an 8-byte setup packet.

There are four types of control commands:

Device Requests
Interface Requests
Endpoint Requests
Hub Requests

*/

#include <stdio.h>
#include <stdint.h>

typedef uint8_t u8;
typedef uint16_t u16;

typedef struct {
	// type of request and recipient
	u8 bmRequestType;

	// The command to be executed. The values of bRequest are listed in the descriptions of the request types.
	u8 bRequest;

	// Command parameter, if needed.
	u16 wValue;

	// Command parameter, if needed.
	u16 wIndex;

	// Number of additional bytes to transfer if the instruction has a data phase.
	u16 wLength;
} SETUP_PACKET;

typedef struct _USB_DEVICE_DESCRIPTOR {
	u8 bLength;
	u8 bDescriptorType;
	u16 bcdUSB;
	u8 bDeviceClass;
	u8 bDeviceSubClass;
	u8 bDeviceProtocol;
	u8 bMaxPacketSize0;
	u16 idVendor;
	u16 idProduct;
	u16 bcdDevice;
	u8 iManufacturer;
	u8 iProduct;
	u8 iSerialNumber;
	u8 bNumConfigurations;
} USB_DEVICE_DESCRIPTOR;

typedef struct _USB_CONFIGURATION_DESCRIPTOR {
	u8 bLength;
	u8 bDescriptorType;
	u16 wTotalLength;
	u8 bNumInterfaces;
	u8 bConfigurationValue;
	u8 iConfiguration;
	u8 bmAttributes;
	u8 MaxPower;
} USB_CONFIGURATION_DESCRIPTOR, *PUSB_CONFIGURATION_DESCRIPTOR;

typedef struct _USB_INTERFACE_DESCRIPTOR {
	u8 bLength;
	u8 bDescriptorType;
	u8 bInterfaceNumber;
	u8 bAlternateSetting;
	u8 bNumEndpoints;
	u8 bInterfaceClass;
	u8 bInterfaceSubClass;
	u8 bInterfaceProtocol;
	u8 iInterface;
} USB_INTERFACE_DESCRIPTOR, *PUSB_INTERFACE_DESCRIPTOR;

typedef struct _USB_ENDPOINT_DESCRIPTOR {
	u8 bLength;
	u8 bDescriptorType;
	u8 bEndpointAddress;
	u8 bmAttributes;
	u16 wMaxPacketSize;
	u8 bInterval;
} USB_ENDPOINT_DESCRIPTOR, *PUSB_ENDPOINT_DESCRIPTOR;

typedef struct _USB_HUB_DESCRIPTOR {
	u8 bDescriptorLength;
	u8 bDescriptorType;
	u8 bNumberOfPorts;
	u16 wHubCharacteristics;
	u8 bPowerOnToPowerGood;
	u8 bHubControlCurrent;
	u8 bRemoveAndPowerMask[64]; // variable length
} USB_HUB_DESCRIPTOR, *PUSB_HUB_DESCRIPTOR;

typedef struct _USB_DEVICE_QUALIFIER_DESCRIPTOR {
	u16 bLength;
	u16 bDescriptorType;
	u16 bcdUSB;
	u16 bDeviceClass;
	u16 bDeviceSubClass;
	u16 bDeviceProtocol;
	u16 bMaxPacketSize0;
	u16 bNumConfigurations;
	u16 bReserved;
} USB_DEVICE_QUALIFIER_DESCRIPTOR, *PUSB_DEVICE_QUALIFIER_DESCRIPTOR;

typedef struct _USB_STRING_DESCRIPTOR {
	u8 bLength;
	u8 bDescriptorType;
	u16 bString[1];
} USB_STRING_DESCRIPTOR, *PUSB_STRING_DESCRIPTOR;

typedef enum _USB_DEVICE_SPEED {
	UsbLowSpeed,  // 1.5 mbit
	UsbFullSpeed, // 12 mbit
	UsbHighSpeed, // 480 mbit
	UsbSuperSpeed // 5 gbit
} USB_DEVICE_SPEED;

int
main()
{
	return 0;
}
