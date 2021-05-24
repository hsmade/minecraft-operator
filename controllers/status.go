package controllers

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/go-mc/mcping"
	v1 "github.com/hsmade/minecraft-operator/api/v1"
	"github.com/hsmade/minecraft-operator/loglevels"
	"github.com/pkg/errors"
	"net"
	"time"
)

// UpdateStatus updates the Server status
func (r *ServerReconciler) UpdateStatus(ctx context.Context, log logr.Logger, server *v1.Server) error {
	log.V(loglevels.Verbose).Info("start reconciling of status")

	server.Status.Running = false
	server.Status.LastPong = 0
	server.Status.Players = []string{}

	if !server.Spec.Enabled {
		log.V(loglevels.Flow).Info("server disabled, adjusting status")

		log.V(loglevels.Verbose).Info("storing status")
		log.V(loglevels.Trace).Info("server status", "status", server.Status)
		err := r.Status().Update(ctx, server)
		if err != nil {
			return errors.Wrap(err, "storing status")
		}
	}

	log.V(loglevels.Verbose).Info("pinging server")
	log.V(loglevels.Flow).Info("connecting to service")
	dest := fmt.Sprintf("%s.%s.svc.cluster.local:25565", server.Name, server.Namespace)
	log.V(loglevels.Trace).Info("connecting", "dest", dest)
	conn, err := net.Dial("tcp", dest)
	if err != nil {
		log.V(loglevels.Info).Info("could not connect to server", "error", err)

		log.V(loglevels.Verbose).Info("storing status")
		log.V(loglevels.Trace).Info("server status", "status", server.Status)
		err := r.Status().Update(ctx, server)
		if err != nil {
			return errors.Wrap(err, "storing status")
		}
		return nil
	}
	log.V(loglevels.Flow).Info("connected to service ok")

	log.V(loglevels.Flow).Info("pinging server")
	status, _, err := mcping.PingAndListConn(conn, 578)
	if err != nil {
		log.V(loglevels.Info).Info("could not ping server", "error", err)

		log.V(loglevels.Verbose).Info("storing status")
		log.V(loglevels.Trace).Info("server status", "status", server.Status)
		err := r.Status().Update(ctx, server)
		if err != nil {
			return errors.Wrap(err, "storing status")
		}
		return nil
	}
	log.V(loglevels.Flow).Info("pinged server ok")
	log.V(loglevels.Trace).Info("server ping result", "status", status)

	log.V(loglevels.Flow).Info("looping over found players sample to add to status")
	server.Status.Players = []string{}
	for _, player := range status.Players.Sample {
		server.Status.Players = append(server.Status.Players, player.Name)
	}
	log.V(loglevels.Trace).Info("players found", "players", server.Status.Players)

	if len(server.Status.Players) > 0 {
		log.V(loglevels.Verbose).Info("updating idle time to now")
		server.Status.IdleTime = time.Now().Unix()
	}

	server.Status.LastPong = time.Now().Unix()
	server.Status.Running = true

	log.V(loglevels.Flow).Info("getting thumbnail from server status")
	icon, err := status.Favicon.ToPNG()
	if err != nil {
		log.V(loglevels.Info).Info("could not get thumbnail from server", "error", err)
	} else {
		server.Status.Thumbnail = base64.StdEncoding.EncodeToString(icon)
		log.V(loglevels.Trace).Info("stored thumbnail", "thumbnail", server.Status.Thumbnail)
	}

	log.V(loglevels.Verbose).Info("storing status")
	log.V(loglevels.Trace).Info("server status", "status", server.Status)
	err = r.Status().Update(ctx, server)
	if err != nil {
		return errors.Wrap(err, "storing status")
	}

	return nil
}
