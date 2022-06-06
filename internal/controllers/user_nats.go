package controllers

import (
	"github.com/cyverse-de/go-mod/gotelnats"
	"github.com/cyverse-de/go-mod/pbinit"
	"github.com/cyverse-de/p/go/qms"
	"github.com/cyverse-de/p/go/svcerror"
	"github.com/cyverse/QMS/internal/db"
	"github.com/sirupsen/logrus"
)

// GetUserOveragesNATS is the NATS handler for listing all of the resources that a user
// is in overage for.
func (s Server) GetUserOveragesNATS(subject, reply string, request *qms.AllUserOveragesRequest) {
	var err error

	log := log.WithFields(logrus.Fields{"context": "list overages"})

	responseList := pbinit.NewOverageList()
	ctx, span := pbinit.InitAllUserOveragesRequest(request, subject)
	defer span.End()

	username := request.Username

	results, err := db.GetUserOverages(ctx, s.GORMDB, username)
	if err != nil {
		responseList.Error = gotelnats.InitServiceError(
			ctx, err, &gotelnats.ErrorOptions{
				ErrorCode: svcerror.ErrorCode_INTERNAL,
			},
		)

	}
	log.Debug("after calling db.GetUserOverages()")

	if results != nil {
		for _, r := range results {
			responseList.Overages = append(responseList.Overages, &qms.Overage{
				ResourceName: r["resource_type_name"].(string),
				Quota:        r["quota"].(float32),
				Usage:        r["usage"].(float32),
			})
		}
	}

	if err = gotelnats.PublishResponse(ctx, s.NATSConn, reply, responseList); err != nil {
		log.Error(err)
	}
}

// InOverageNATS is the NATS handler for checking if a user is in overage
// for a particular resource type.
func (s Server) InOverageNATS(subject, reply string, request *qms.IsOverageRequest) {
	log := log.WithFields(logrus.Fields{"context": "check if in overage"})

	response := pbinit.NewIsOverage()
	ctx, span := pbinit.InitIsOverageRequest(request, subject)
	defer span.End()

	results, err := db.IsOverage(ctx, s.GORMDB, request.GetUsername(), request.GetResourceName())
	if err != nil {
		response.Error = gotelnats.InitServiceError(
			ctx, err, &gotelnats.ErrorOptions{
				ErrorCode: svcerror.ErrorCode_INTERNAL,
			},
		)
	}

	log.Debug("after calling db.IsOverage()")
	log.Debugf("results are %+v\n", results)

	if results != nil {
		response.IsOverage = results["overage"].(bool)
	}

	if err = gotelnats.PublishResponse(ctx, s.NATSConn, reply, response); err != nil {
		log.Error(err)
	}
}
