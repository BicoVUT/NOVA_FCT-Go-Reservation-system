///////////////////////////////////////////////////////////////////////
//////////////// Simple Reservations System (Task 2) //////////////////
///////////////////////////////////////////////////////////////////////

// Summary: The base is the same message-style designed system as in
//          task 1. The main difference is that the users now have
//          a priority. VIP users can overwrite non-VIP users bookings
//          but not the other way around.

///////////////////////////////////////////////////////////////////////

package main

import (
	"fmt"
	"strconv"
	"time"
)

//////////////////// Definition of useful Structs ////////////////////

// struct for user
type User struct {
	Id       int
	Priority bool
	inbox    chan BookingMessage
}

// struct of list of users
type Users struct {
	Users []User
}

// Struct for Message to the Users inboxes
// action 0: booking successful
// action 1: booking not successful
// action 2: booking cancelled
type BookingMessage struct {
	Booking Booking
	Action  int
	Message string
}

// Booking messages can be collected and then be sent out
type BookingMessages struct {
	BookingMessages []BookingMessage
}

// struct for booking
type Booking struct {
	User       User
	FacilityID string
	StartTime  int
	EndTime    int
}

// struct for facilities
type Facility struct {
	Id       string // like "room" or "projector"
	Bookings []Booking
	Capacity int
	Inbox    chan Request
}

// struct for all facilities
type Facilities struct {
	Facilities []Facility
}

// struct for request, send to facility
type Request struct {
	FacilityId string
	Booking    Booking
}

// time struct
type ProgramTime struct {
	time int
}

// //////////////////// Time functionality //////////////////////
// get program time
func (pt *ProgramTime) GetCurrentTime() int {
	return pt.time
}

// start program time
func StartProgramTime() *ProgramTime {
	// receives current time
	ticker := time.NewTicker(time.Second)

	// store program time
	var programTime ProgramTime

	// increment program time
	go func() {
		for range ticker.C {
			programTime.time++
		}
	}()

	return &programTime
}

// //////////////////// Helpers //////////////////////

func CheckTimeslotOverlap(a1 int, a2 int, b1 int, b2 int) bool {
	return (a1 >= b1 && a1 < b2) || (a2 > b1 && a2 <= b2)
}

// //////////////////// Output //////////////////////

func VIPToString(vip bool) string {
	if vip {
		return "VIP"
	} else {
		return "non-VIP"
	}
}

func ActionToString(action int) string {
	switch action {
	case 0:
		return "Booking successful."
	case 1:
		return "Booking not successful."
	case 2:
		return "Booking cancelled."
	default:
		return "Unknown action."
	}
}

func BookingToString(booking Booking) string {
	// Booking for user X for facility Y from time A to time B.
	return ("Booking for user " + strconv.Itoa(booking.User.Id) + " for facility '" + booking.FacilityID + "' from time " + strconv.Itoa(booking.StartTime) + " to time " + strconv.Itoa(booking.EndTime) + ".")
}

func PrintBookingMessage(message BookingMessage) {
	// Booking for user X for facility Y from time A to time B.
	fmt.Printf("\n====== Booking Message ======\nSubject: %s\nAction: %s\nYour status: %s\nMessage: %s\n============================\n\n", BookingToString(message.Booking), ActionToString(message.Action), VIPToString(message.Booking.User.Priority), message.Message)
}

//////////////////////// Server ////////////////////////

// A server for each facility, so for a specific booking
// only e. g. the "projector department" has to be contacted
func FacilityServer(programTime *ProgramTime, facility *Facility) {
	// wait for bookings
	for {
		// receive booking

		request := <-facility.Inbox

		// check if booking is possible
		messages := CheckBooking(request, programTime, facility)

		// send all messages to users
		for _, message := range messages.BookingMessages {
			message.Booking.User.inbox <- message
		}
	}
}

// Start all facility servers
func StartFacilityServers(programTime *ProgramTime, facilities *Facilities) {
	for i := 0; i < len(facilities.Facilities); i++ {
		go FacilityServer(programTime, &facilities.Facilities[i])
	}
}

// //////////////////// Check Booking //////////////////////

// check if a booking is possible for the given facility, return a list of messages
// for the users affected (list is of either size 1 or 2 depending on whether a
// cancellation occurs)
func CheckBooking(request Request, programTime *ProgramTime, facility *Facility) BookingMessages {
	// check if the booking is in the future, VIPs and non VIPs can
	// only book for timeslots in the future
	if request.Booking.StartTime < programTime.GetCurrentTime() {
		return BookingMessages{BookingMessages: []BookingMessage{{Booking: request.Booking, Action: 1, Message: "Booking is in the past."}}}
	}
	// check if start time is before end time
	if request.Booking.StartTime >= request.Booking.EndTime {
		return BookingMessages{BookingMessages: []BookingMessage{{Booking: request.Booking, Action: 1, Message: "Start time is after end time."}}}
	}

	// go through all bookings of this facility
	num_reserved := 0
	num_reserved_vip := 0
	for _, booking := range facility.Bookings {
		// check if request start time or end time is in the
		// interval of the booking
		if CheckTimeslotOverlap(request.Booking.StartTime, request.Booking.EndTime, booking.StartTime, booking.EndTime) {
			num_reserved++
			if booking.User.Priority {
				num_reserved_vip++
			}
		}
		// check if num_reserved matches capacity
		if (num_reserved >= facility.Capacity && !request.Booking.User.Priority) || (num_reserved_vip >= facility.Capacity) {
			return BookingMessages{BookingMessages: []BookingMessage{{Booking: request.Booking, Action: 1, Message: "Capacity exceeded."}}}
		}
	}
	if num_reserved < facility.Capacity {

		// make booking
		facility.Bookings = append(facility.Bookings, request.Booking)

		return BookingMessages{BookingMessages: []BookingMessage{{Booking: request.Booking, Action: 0, Message: "Facilities were booked."}}}
	}
	// the only possible case here is that the VIP has to overbook a non-VIP
	// thus remove one non-VIP booking and add the VIP booking
	for i, booking := range facility.Bookings {
		if CheckTimeslotOverlap(request.Booking.StartTime, request.Booking.EndTime, booking.StartTime, booking.EndTime) && !booking.User.Priority {
			// delete booking of non vip and add cancellation message to non-vip to message list
			facility.Bookings = append(facility.Bookings[:i], facility.Bookings[i+1:]...) // delete in the for loop is ok as we break afterwards
			booking_messages := BookingMessages{BookingMessages: []BookingMessage{{Booking: booking, Action: 2, Message: "A VIP-User has overwritten your booking."}}}
			// add vip booking and add vip booking message to message list
			facility.Bookings = append(facility.Bookings, request.Booking)
			booking_messages.BookingMessages = append(booking_messages.BookingMessages, BookingMessage{Booking: request.Booking, Action: 0, Message: "Facilities were booked, you have overwritten a non-VIP booking."})
			return booking_messages
		}
	}
	return BookingMessages{BookingMessages: []BookingMessage{{Booking: request.Booking, Action: 1, Message: "Facility not found."}}}
}

// // //////////// Make Booking ////////////////////
// sends a booking request to the facility
func MakeBooking(user User, facility *Facility, StartTime int, EndTime int) {
	Booking := Booking{User: user, FacilityID: facility.Id, StartTime: StartTime, EndTime: EndTime}
	facility.Inbox <- Request{FacilityId: facility.Id, Booking: Booking}
}

// //////////////////// User //////////////////////

func StartBooker(user User, facilities *Facilities, start_times []int, end_times []int) {
	// do some bookings
	go func() {
		for i := 0; i < len(start_times); i++ {
			go MakeBooking(user, &facilities.Facilities[i], start_times[i], end_times[i])
		}
	}()

	for {
		// receive message
		message := <-user.inbox

		// print message
		PrintBookingMessage(message)
	}
}

// //////////////////// Main //////////////////////

func main() {

	// start program time
	programTime := StartProgramTime()
	fmt.Println("=========== Program started ===========")

	// create facilities
	rooms := Facility{Id: "room", Capacity: 2, Inbox: make(chan Request)}
	projectors := Facility{Id: "projector", Capacity: 2, Inbox: make(chan Request)}

	// create facilities struct
	facilities := Facilities{Facilities: []Facility{rooms, projectors}}

	// create users
	user1 := User{Id: 1, Priority: false, inbox: make(chan BookingMessage)}
	user2 := User{Id: 2, Priority: false, inbox: make(chan BookingMessage)}
	user3 := User{Id: 3, Priority: true, inbox: make(chan BookingMessage)}
	user4 := User{Id: 4, Priority: true, inbox: make(chan BookingMessage)}
	user5 := User{Id: 5, Priority: true, inbox: make(chan BookingMessage)}

	// start server
	go StartFacilityServers(programTime, &facilities)

	/////////////////////// Simple Test ///////////////////////
	// all users try to book a room in the first and a projector in the second time specified
	go StartBooker(user1, &facilities, []int{10, 10}, []int{15, 15})
	go StartBooker(user2, &facilities, []int{10, 10}, []int{15, 15})
	go StartBooker(user3, &facilities, []int{10, 10}, []int{15, 15})
	go StartBooker(user4, &facilities, []int{10, 10}, []int{15, 15})
	go StartBooker(user5, &facilities, []int{10, 15}, []int{15, 20})

	// Expected Behavior:
	// at the end vip users should have should have all rooms and projectors from 10 to 15
	// one vip user is declined a room as of capacity exceeded, no vip user is declined a projector
	// non-vip users are either overwritten or declined depending on the race

	///////////////////////////////////////////////////////////

	// very simple "keep-alive-system"
	// let program run until program time is 100
	for programTime.GetCurrentTime() < 100 {
		time.Sleep(1 * time.Second)
	}
}

////////////////////// TODO //////////////////////

// Improvements: - more specific tests
