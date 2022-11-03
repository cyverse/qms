package controllers

import (
	"strings"

	"github.com/cockroachdb/apd"
	"github.com/cyverse-de/go-mod/gotelnats"
	"github.com/cyverse-de/go-mod/pbinit"
	"github.com/cyverse-de/p/go/qms"
	"github.com/cyverse-de/p/go/svcerror"
	"github.com/cyverse/QMS/internal/db"
	"github.com/sirupsen/logrus"
)

func parseFloat64(floatStr string) (float64, error) {
	d, _, err := apd.New(0, 0).SetString(floatStr)
	if err != nil {
		return 0.0, err
	}

	f, err := d.Float64()
	if err != nil {
		return 0.0, err
	}

	return f, nil
}

// InOverageNATS is the NATS handler for checking if a user is in overage
// for a particular resource type.
func (s Server) InOverageNATS(subject, reply string, request *qms.IsOverageRequest) {
	var err error

	log := log.WithFields(logrus.Fields{"context": "check if in overage"})

	response := pbinit.NewIsOverage()
	ctx, span := pbinit.InitIsOverageRequest(request, subject)
	defer span.End()

	// Always return false if s.ReportOverages is false.
	if !s.ReportOverages {
		response.IsOverage = false

		if err = gotelnats.PublishResponse(ctx, s.NATSConn, reply, response); err != nil {
			log.Error(err)
		}

		return
	}

	username := strings.TrimSuffix(request.GetUsername(), s.UsernameSuffix)
	results, err := db.IsOverage(ctx, s.GORMDB, username, request.GetResourceName())
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
