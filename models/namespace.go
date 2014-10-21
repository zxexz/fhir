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

type Namespace struct {
	Id          string                       `json:"-" bson:"_id"`
	Type        string                       `bson:"type"`
	Name        string                       `bson:"name"`
	Status      string                       `bson:"status"`
	Country     string                       `bson:"country"`
	Category    CodeableConcept              `bson:"category"`
	Responsible string                       `bson:"responsible"`
	Description string                       `bson:"description"`
	Usage       string                       `bson:"usage"`
	UniqueId    []NamespaceUniqueIdComponent `bson:"uniqueId"`
	Contact     NamespaceContactComponent    `bson:"contact"`
	ReplacedBy  Reference                    `bson:"replacedBy"`
}

// This is an ugly hack to deal with embedded structures in the spec uniqueId
type NamespaceUniqueIdComponent struct {
	Type      string `bson:"type"`
	Value     string `bson:"value"`
	Preferred bool   `bson:"preferred"`
	Period    Period `bson:"period"`
}

// This is an ugly hack to deal with embedded structures in the spec contact
type NamespaceContactComponent struct {
	Name    HumanName      `bson:"name"`
	Telecom []ContactPoint `bson:"telecom"`
}
