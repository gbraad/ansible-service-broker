//
// Copyright (c) 2018 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Red Hat trademarks are not licensed under Apache License, Version 2.
// No permission is granted to use or replicate Red Hat trademarks that
// are incorporated in this software or its documentation.
//

package broker

import (
	"github.com/openshift/ansible-service-broker/pkg/apb"
	"github.com/openshift/ansible-service-broker/pkg/metrics"
	"github.com/pborman/uuid"
)

// BindingJob - Job to provision
type BindingJob struct {
	serviceInstance *apb.ServiceInstance
	bindingUUID     uuid.UUID
	params          *apb.Parameters
	bind            apb.Binder
}

// NewBindingJob - Create a new binding job.
func NewBindingJob(serviceInstance *apb.ServiceInstance, bindingUUID uuid.UUID, params *apb.Parameters, bind apb.Binder) *BindingJob {
	return &BindingJob{
		serviceInstance: serviceInstance,
		bindingUUID:     bindingUUID,
		params:          params,
		bind:            bind,
	}
}

// Run - run the binding job.
func (p *BindingJob) Run(token string, msgBuffer chan<- JobMsg) {
	metrics.BindingJobStarted()
	defer metrics.BindingJobFinished()
	jobMsg := JobMsg{
		InstanceUUID: p.serviceInstance.ID.String(),
		JobToken:     token,
		SpecID:       p.serviceInstance.Spec.ID,
		BindingUUID:  p.bindingUUID.String(),
		State: apb.JobState{
			State:  apb.StateInProgress,
			Method: apb.JobMethodBind,
			Token:  token,
		},
	}
	log.Debug("bindjob: binding job started, calling apb.Bind")

	msgBuffer <- jobMsg

	podName, extCreds, err := p.bind(p.serviceInstance, p.params)

	log.Debug("bindjob: returned from apb.Bind")

	if err != nil {
		log.Errorf("bindjob::Binding error occurred.\n%s", err.Error())
		jobMsg.State.State = apb.StateFailed
		errMsg := "Error occurred during binding. Please contact administrator if it persists."
		// Because we know the error we should return that error.
		if err == apb.ErrorPodPullErr {
			errMsg = err.Error()
		}

		// send error message
		// can't have an error type in a struct you want marshalled
		// https://github.com/golang/go/issues/5161
		jobMsg.State.Error = errMsg
		msgBuffer <- jobMsg
		return
	}

	// send creds
	log.Debug("bindjob: looks like we're done, sending credentials")
	if nil != extCreds {
		jobMsg.ExtractedCredentials = *extCreds
	}
	jobMsg.State.State = apb.StateSucceeded
	jobMsg.PodName = podName
	msgBuffer <- jobMsg
}
