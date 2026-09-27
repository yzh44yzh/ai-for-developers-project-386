package domain

import "github.com/google/uuid"

// NewUserID generates a new unique UserID.
func NewUserID() UserID { return UserID(uuid.NewString()) }

// NewMeetingID generates a new unique MeetingID.
func NewMeetingID() MeetingID { return MeetingID(uuid.NewString()) }
