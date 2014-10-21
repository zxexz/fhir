// Copyright (c) 2011-2014, HL7, Inc & The MITRE Corporation
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
//     * Redistributions of source code must retain the above copyright notice, this
//       list of conditions and the following disclaimer.
//     * Redistributions in binary form must reproduce the above copyright notice,
//       this list of conditions and the following disclaimer in the documentation
//       and/or other materials provided with the distribution.
//     * Neither the name of HL7 nor the names of its contributors may be used to
//       endorse or promote products derived from this software without specific
//       prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED.
// IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT,
// INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT
// NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR
// PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY,
// WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
// ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
// POSSIBILITY OF SUCH DAMAGE.

package models

import "time"

type Specimen struct {
	Id                  string                       `json:"-" bson:"_id"`
	Identifier          []Identifier                 `bson:"identifier"`
	Type                CodeableConcept              `bson:"type"`
	Source              []SpecimenSourceComponent    `bson:"source"`
	Subject             Reference                    `bson:"subject"`
	AccessionIdentifier Identifier                   `bson:"accessionIdentifier"`
	ReceivedTime        time.Time                    `bson:"receivedTime"`
	Collection          SpecimenCollectionComponent  `bson:"collection"`
	Treatment           []SpecimenTreatmentComponent `bson:"treatment"`
	Container           []SpecimenContainerComponent `bson:"container"`
}

// This is an ugly hack to deal with embedded structures in the spec source
type SpecimenSourceComponent struct {
	Relationship string      `bson:"relationship"`
	Target       []Reference `bson:"target"`
}

// This is an ugly hack to deal with embedded structures in the spec collection
type SpecimenCollectionComponent struct {
	Collector         Reference       `bson:"collector"`
	Comment           []string        `bson:"comment"`
	CollectedDateTime time.Time       `bson:"collectedDateTime"`
	CollectedPeriod   Period          `bson:"collectedPeriod"`
	Quantity          Quantity        `bson:"quantity"`
	Method            CodeableConcept `bson:"method"`
	SourceSite        CodeableConcept `bson:"sourceSite"`
}

// This is an ugly hack to deal with embedded structures in the spec treatment
type SpecimenTreatmentComponent struct {
	Description string          `bson:"description"`
	Procedure   CodeableConcept `bson:"procedure"`
	Additive    []Reference     `bson:"additive"`
}

// This is an ugly hack to deal with embedded structures in the spec container
type SpecimenContainerComponent struct {
	Identifier       []Identifier    `bson:"identifier"`
	Description      string          `bson:"description"`
	Type             CodeableConcept `bson:"type"`
	Capacity         Quantity        `bson:"capacity"`
	SpecimenQuantity Quantity        `bson:"specimenQuantity"`
	Additive         Reference       `bson:"additive"`
}
