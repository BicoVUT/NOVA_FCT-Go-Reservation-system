///////////////////////////////////////////////////////////////////////
//////////////// Simple Reservations System (Task 3) //////////////////
///////////////////////////////////////////////////////////////////////

// Summary: In this system compounds of facilities can be booked.
//          Here, conflicts in compound booking are not allowed
//          (see the legacy version which enables this).
//          The different requests making up a compound booking
//          are sent to the server concurrently, compounds as a whole
//          are handled sequentially.

///////////////////////////////////////////////////////////////////////

package main

import (
	"fmt"
	"math/rand"
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

// actions (add booking or remove booking)
type Action struct {
	Action   int // 0: add booking, 1: remove booking
	Booking  Booking
	Facility *Facility
}

// struct of list of actions
type Actions struct {
	Actions []Action
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

// Response with actions and messages
type CheckResponse struct {
	Actions         Actions
	BookingMessages BookingMessages
}

// struct for booking
type Booking struct {
	User        User
	FacilityID  string
	StartTime   int
	EndTime     int
	RequestsPtr *Requests
}

// struct for facilities
type Facility struct {
	Id       string // like "room" or "projector"
	Bookings []Booking
	Capacity int
}

// struct for all facilities
type Facilities struct {
	Facilities []Facility
}

// request struct
type Request struct {
	Facility *Facility
	Booking  Booking
}

type Requests struct {
	Requests []Request
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
func GenRandTimeslot() (int, int) {
	a := rand.Intn(101)
	return a, a + 10
}

// function to check if two bookings are the same
func CheckBookingEquality(a Booking, b Booking) bool {
	return a.User.Id == b.User.Id && a.FacilityID == b.FacilityID && a.StartTime == b.StartTime && a.EndTime == b.EndTime
}

func CheckTimeslotOverlap(a1 int, a2 int, b1 int, b2 int) bool {
	return (a1 >= b1 && a1 < b2) || (a2 > b1 && a2 <= b2)
}

func AddBooking(booking Booking, facility *Facility) {
	facility.Bookings = append(facility.Bookings, booking)
}

func RemoveBooking(booking Booking, facility *Facility) {
	for i, b := range facility.Bookings {
		if CheckBookingEquality(b, booking) {
			facility.Bookings = append(facility.Bookings[:i], facility.Bookings[i+1:]...)
			break
		}
	}
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

// server with input channel taking a requests, requests must consist of different ressources
// for the ability of room-room bookings (e. g. when I have a big meeting and need two rooms)
// concurrent handling as done here is not possible, use the legacy version in this case

func Server(input chan Requests, programTime *ProgramTime, facilities *Facilities) {
	// infinite loop
	for {
		// receive booking
		// compounds cannot be booked so that the parts interlace as this would change
		// the validity
		// each compound as a whole has to be done one after the other
		// still the parts of the compound can be done concurrently
		// assuming compounds can only include different ressources and not e. g. two rooms

		requests := <-input

		messages := BookingMessages{BookingMessages: []BookingMessage{}}
		actions := Actions{Actions: []Action{}}

		compound_possible := true

		// create a channel for CheckResponse
		check_response := make(chan CheckResponse) // could also be allocated only once before the for loop

		// assuming the compounds cannot include the same facility twice
		for _, request := range requests.Requests {
			request.Booking.RequestsPtr = &requests
			go CheckBooking(request, programTime, request.Facility, check_response)
		}

		// receive all responses
		for i := 0; i < len(requests.Requests); i++ {
			check_response := <-check_response
			m, a := check_response.BookingMessages, check_response.Actions
			// add messages if not already in there
			for _, m := range m.BookingMessages {
				already_in := false
				for _, message := range messages.BookingMessages {
					if CheckBookingEquality(message.Booking, m.Booking) {
						already_in = true
						break
					}
				}
				if !already_in {
					messages.BookingMessages = append(messages.BookingMessages, m)
				}
			}
			// add actions if not already in there
			for _, a := range a.Actions {
				already_in := false
				for _, action := range actions.Actions {
					if CheckBookingEquality(action.Booking, a.Booking) {
						already_in = true
						break
					}
				}
				if !already_in {
					actions.Actions = append(actions.Actions, a)
				}
			}
		}

		for _, message := range messages.BookingMessages {
			if message.Action == 1 {
				compound_possible = false
			}
		}

		if compound_possible {
			// execute actions
			for _, action := range actions.Actions {
				switch action.Action {
				case 0:
					AddBooking(action.Booking, action.Facility)
				case 1:
					RemoveBooking(action.Booking, action.Facility)
				default:
					fmt.Println("Unknown action.")
				}
			}

			// send all messages to users
			for _, message := range messages.BookingMessages {
				message.Booking.User.inbox <- message
			}
		} else {
			// send all messages to users
			for _, message := range messages.BookingMessages {
				if message.Action != 2 {
					message.Booking.User.inbox <- BookingMessage{Booking: message.Booking, Action: 1, Message: "Booking not successful as a part of the compound could not be booked."}
				}
			}
		}
	}
}

// //////////////////// Check Booking //////////////////////

// check if a booking is possible for the given facility, return a list of messages
// for the users affected (list is of either size 1 or 2 depending on whether a
// cancellation occurs)
func CheckBooking(request Request, programTime *ProgramTime, facility *Facility, ResponseChannel chan CheckResponse) {
	// check if the booking is in the future, VIPs and non VIPs can
	// only book for timeslots in the future
	Actions := Actions{Actions: []Action{}}
	if request.Booking.StartTime < programTime.GetCurrentTime() {
		bm, a := BookingMessages{BookingMessages: []BookingMessage{{Booking: request.Booking, Action: 1, Message: "Booking is in the past."}}}, Actions
		ResponseChannel <- CheckResponse{BookingMessages: bm, Actions: a}
		return
	}
	// check if start time is before end time
	if request.Booking.StartTime >= request.Booking.EndTime {
		bm, a := BookingMessages{BookingMessages: []BookingMessage{{Booking: request.Booking, Action: 1, Message: "Start time is after end time."}}}, Actions
		ResponseChannel <- CheckResponse{BookingMessages: bm, Actions: a}
		return
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
			bm, a := BookingMessages{BookingMessages: []BookingMessage{{Booking: request.Booking, Action: 1, Message: "Capacity exceeded."}}}, Actions
			ResponseChannel <- CheckResponse{BookingMessages: bm, Actions: a}
			return
		}
	}
	if num_reserved < facility.Capacity {

		// make booking
		Actions.Actions = append(Actions.Actions, Action{Action: 0, Booking: request.Booking, Facility: facility})

		bm, a := BookingMessages{BookingMessages: []BookingMessage{{Booking: request.Booking, Action: 0, Message: "Facilities were booked."}}}, Actions
		ResponseChannel <- CheckResponse{BookingMessages: bm, Actions: a}
		return
	}
	// the only possible case here is that the VIP has to overbook a non-VIP
	// thus remove one non-VIP booking and add the VIP booking
	for _, booking := range facility.Bookings {
		if CheckTimeslotOverlap(request.Booking.StartTime, request.Booking.EndTime, booking.StartTime, booking.EndTime) && !booking.User.Priority {
			// delete booking of non vip and add cancellation message to non-vip to message list
			booking_messages := BookingMessages{BookingMessages: []BookingMessage{}}

			for _, requestt := range (*booking.RequestsPtr).Requests {
				Actions.Actions = append(Actions.Actions, Action{Action: 1, Booking: requestt.Booking, Facility: requestt.Facility})
				booking_messages.BookingMessages = append(booking_messages.BookingMessages, BookingMessage{Booking: requestt.Booking, Action: 2, Message: "A VIP-User has overwritten your booking."})
			}

			// add vip booking and add vip booking message to message list
			Actions.Actions = append(Actions.Actions, Action{Action: 0, Booking: request.Booking, Facility: facility})
			booking_messages.BookingMessages = append(booking_messages.BookingMessages, BookingMessage{Booking: request.Booking, Action: 0, Message: "Facilities were booked, you have overwritten a non-VIP booking."})
			bm, a := booking_messages, Actions
			ResponseChannel <- CheckResponse{BookingMessages: bm, Actions: a}
			return
		}
	}

	bm, a := BookingMessages{BookingMessages: []BookingMessage{{Booking: request.Booking, Action: 1, Message: "Facility not found."}}}, Actions
	ResponseChannel <- CheckResponse{BookingMessages: bm, Actions: a}
	return
}

// // //////////// Make Booking ////////////////////
func MakeBookings(user User, facilities *Facilities, StartTime int, EndTime int, input chan Requests) {
	Requests := Requests{Requests: []Request{}}
	for i, _ := range facilities.Facilities {
		facility := &facilities.Facilities[i]
		Requests.Requests = append(Requests.Requests, Request{Facility: facility, Booking: Booking{User: user, FacilityID: facility.Id, StartTime: StartTime, EndTime: EndTime}})
	}
	input <- Requests
}

// //////////////////// User //////////////////////

func StartBooker(user User, input chan Requests, facilities *Facilities, start_times []int, end_times []int) {
	// do some bookings
	go func() {
		for i := 0; i < len(start_times); i++ {
			if user.Priority {
				time.Sleep(2 * time.Second)
			} else {
				time.Sleep(1 * time.Second)
			}
			go MakeBookings(user, facilities, start_times[i], end_times[i], input)
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
	rooms := Facility{Id: "room", Capacity: 2}
	projectors := Facility{Id: "projector", Capacity: 2}

	// create facilities struct
	facilities := Facilities{Facilities: []Facility{rooms, projectors}}

	// create users
	user1 := User{Id: 1, Priority: false, inbox: make(chan BookingMessage)}
	user2 := User{Id: 2, Priority: false, inbox: make(chan BookingMessage)}
	user3 := User{Id: 3, Priority: true, inbox: make(chan BookingMessage)}
	user4 := User{Id: 4, Priority: true, inbox: make(chan BookingMessage)}
	user5 := User{Id: 5, Priority: true, inbox: make(chan BookingMessage)}

	// create input channel
	input := make(chan Requests)

	// start server
	go Server(input, programTime, &facilities)

	// mind prints are not in order as the program is concurrent

	/////////////////////// Simple Test ///////////////////////
	// Test cancellation mechanism - works :)
	go StartBooker(user1, input, &facilities, []int{10}, []int{15})
	go StartBooker(user2, input, &facilities, []int{10}, []int{15})
	go StartBooker(user3, input, &facilities, []int{10}, []int{15}) // VIP

	// Check the case of multiple VIP users, all want to book room and projector
	go StartBooker(user4, input, &facilities, []int{10}, []int{15}) // VIP
	go StartBooker(user5, input, &facilities, []int{10}, []int{15}) // VIP

	// expected behavior: at the end two VIPs will have both room and projector
	// one-VIP will be declined, the non-vips will be declined / cancelled

	///////////////////////////////////////////////////////////

	// very simple "keep-alive-system"
	// let program run until program time is 100
	for programTime.GetCurrentTime() < 100 {
		time.Sleep(1 * time.Second)
	}
}

////////////////////// TODO //////////////////////

// Improvements: - more specific tests
//               - handle bookings as references -> easier comparison, avoid lots of copies
