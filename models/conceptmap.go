// Copyright (c) 2011-2017, HL7, Inc & The MITRE Corporation
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

import (
	"encoding/json"
	"errors"
	"fmt"
)

type ConceptMap struct {
	DomainResource  `bson:",inline"`
	Url             string                     `bson:"url,omitempty" json:"url,omitempty"`
	Identifier      *Identifier                `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Version         string                     `bson:"version,omitempty" json:"version,omitempty"`
	Name            string                     `bson:"name,omitempty" json:"name,omitempty"`
	Title           string                     `bson:"title,omitempty" json:"title,omitempty"`
	Status          string                     `bson:"status,omitempty" json:"status,omitempty"`
	Experimental    *bool                      `bson:"experimental,omitempty" json:"experimental,omitempty"`
	Date            *FHIRDateTime              `bson:"date,omitempty" json:"date,omitempty"`
	Publisher       string                     `bson:"publisher,omitempty" json:"publisher,omitempty"`
	Contact         []ContactDetail            `bson:"contact,omitempty" json:"contact,omitempty"`
	Description     string                     `bson:"description,omitempty" json:"description,omitempty"`
	UseContext      []UsageContext             `bson:"useContext,omitempty" json:"useContext,omitempty"`
	Jurisdiction    []CodeableConcept          `bson:"jurisdiction,omitempty" json:"jurisdiction,omitempty"`
	Purpose         string                     `bson:"purpose,omitempty" json:"purpose,omitempty"`
	Copyright       string                     `bson:"copyright,omitempty" json:"copyright,omitempty"`
	SourceUri       string                     `bson:"sourceUri,omitempty" json:"sourceUri,omitempty"`
	SourceReference *Reference                 `bson:"sourceReference,omitempty" json:"sourceReference,omitempty"`
	TargetUri       string                     `bson:"targetUri,omitempty" json:"targetUri,omitempty"`
	TargetReference *Reference                 `bson:"targetReference,omitempty" json:"targetReference,omitempty"`
	Group           []ConceptMapGroupComponent `bson:"group,omitempty" json:"group,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *ConceptMap) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "ConceptMap"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to ConceptMap), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *ConceptMap) GetBSON() (interface{}, error) {
	x.ResourceType = "ConceptMap"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "conceptMap" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type conceptMap ConceptMap

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *ConceptMap) UnmarshalJSON(data []byte) (err error) {
	x2 := conceptMap{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = ConceptMap(x2)
		return x.checkResourceType()
	}
	return
}

func (x *ConceptMap) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "ConceptMap"
	} else if x.ResourceType != "ConceptMap" {
		return errors.New(fmt.Sprintf("Expected resourceType to be ConceptMap, instead received %s", x.ResourceType))
	}
	return nil
}

type ConceptMapGroupComponent struct {
	BackboneElement `bson:",inline"`
	Source          string                             `bson:"source,omitempty" json:"source,omitempty"`
	SourceVersion   string                             `bson:"sourceVersion,omitempty" json:"sourceVersion,omitempty"`
	Target          string                             `bson:"target,omitempty" json:"target,omitempty"`
	TargetVersion   string                             `bson:"targetVersion,omitempty" json:"targetVersion,omitempty"`
	Element         []ConceptMapSourceElementComponent `bson:"element,omitempty" json:"element,omitempty"`
	Unmapped        *ConceptMapGroupUnmappedComponent  `bson:"unmapped,omitempty" json:"unmapped,omitempty"`
}

type ConceptMapSourceElementComponent struct {
	BackboneElement `bson:",inline"`
	Code            string                             `bson:"code,omitempty" json:"code,omitempty"`
	Display         string                             `bson:"display,omitempty" json:"display,omitempty"`
	Target          []ConceptMapTargetElementComponent `bson:"target,omitempty" json:"target,omitempty"`
}

type ConceptMapTargetElementComponent struct {
	BackboneElement `bson:",inline"`
	Code            string                            `bson:"code,omitempty" json:"code,omitempty"`
	Display         string                            `bson:"display,omitempty" json:"display,omitempty"`
	Equivalence     string                            `bson:"equivalence,omitempty" json:"equivalence,omitempty"`
	Comment         string                            `bson:"comment,omitempty" json:"comment,omitempty"`
	DependsOn       []ConceptMapOtherElementComponent `bson:"dependsOn,omitempty" json:"dependsOn,omitempty"`
	Product         []ConceptMapOtherElementComponent `bson:"product,omitempty" json:"product,omitempty"`
}

type ConceptMapOtherElementComponent struct {
	BackboneElement `bson:",inline"`
	Property        string `bson:"property,omitempty" json:"property,omitempty"`
	System          string `bson:"system,omitempty" json:"system,omitempty"`
	Code            string `bson:"code,omitempty" json:"code,omitempty"`
	Display         string `bson:"display,omitempty" json:"display,omitempty"`
}

type ConceptMapGroupUnmappedComponent struct {
	BackboneElement `bson:",inline"`
	Mode            string `bson:"mode,omitempty" json:"mode,omitempty"`
	Code            string `bson:"code,omitempty" json:"code,omitempty"`
	Display         string `bson:"display,omitempty" json:"display,omitempty"`
	Url             string `bson:"url,omitempty" json:"url,omitempty"`
}
