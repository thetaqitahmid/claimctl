import { useState, useMemo, useEffect, useRef } from "react";
import { useTranslation } from "react-i18next";
import ResourceTable from "./ResourceTable";
import ResourceDetailsModal from "./ResourceDetailsModal";
import ConfirmationModal from "./ConfirmationModal";
import ReservationModal from "./ReservationModal";
import { Resource, ResourceWithStatus, ReservationOptions, Space } from "../types";
import {
  useGetResourcesQuery,
  useDeleteResourceMutation,
  useSetMaintenanceModeMutation,
} from "../store/api/resources";
import {
  useClaimReservationMutation,
  useCancelReservationMutation,
  useCompleteReservationMutation,
  useGetUserReservationsQuery,
  useCancelAllReservationsMutation,
} from "../store/api/reservations";
import { useGetSpacesQuery } from "../store/api/spaces";
import { Box, Plus } from "lucide-react";
import AddResourcePopup from "./AddResource";
import { useCreateResourceMutation } from "../store/api/resources";
import { useAppSelector } from "../store/store";
import { useNotificationContext } from "../hooks/useNotification";
import useSessionState from "../hooks/useSessionState";

const ResourceList = () => {
  const { t } = useTranslation(["components", "pages", "common"]);
  const isAdmin = useAppSelector((state) => state.authSlice.role === "admin");
  const [viewingResource, setViewingResource] = useState<Resource | null>(null);
  const [cancelAllResourceId, setCancelAllResourceId] = useState<string | null>(
    null,
  );
  const [reservationResource, setReservationResource] = useState<{
    id: string;
    name: string;
  } | null>(null);
  const [maintenanceModalResource, setMaintenanceModalResource] = useState<{
    id: string;
    name: string;
    currentState: boolean;
  } | null>(null);
  const [maintenanceReason, setMaintenanceReason] = useState("");
  const { showNotification } = useNotificationContext();

  const {
    data: resourcesWithStatus = [],
    error: apiError,
    isLoading,
    isFetching,
    refetch: refetchResources,
  } = useGetResourcesQuery();
  const { data: spaces } = useGetSpacesQuery();
  const [selectedSpaceId, setSelectedSpaceId] = useSessionState<string | null>(
    "rm_selectedSpaceId",
    null,
  );

  const [deleteResource] = useDeleteResourceMutation();
  const [claimReservation] = useClaimReservationMutation();
  const reservationAttempt = useRef<{ fingerprint: string; key: string } | null>(null);
  const actionsInFlight = useRef(new Set<string>());
  const [busyResourceIds, setBusyResourceIds] = useState<string[]>([]);
  const startAction = (id: string) => {
    if (actionsInFlight.current.has(id)) return false;
    actionsInFlight.current.add(id);
    setBusyResourceIds([...actionsInFlight.current]);
    return true;
  };
  const finishAction = (id: string) => {
    actionsInFlight.current.delete(id);
    setBusyResourceIds([...actionsInFlight.current]);
  };
  const [cancelReservation] = useCancelReservationMutation();
  const [completeReservation] = useCompleteReservationMutation();
  const [cancelAllReservations] = useCancelAllReservationsMutation();
  const { data: userReservations, isLoading: isUserReservationsLoading,
    isFetching: isUserReservationsFetching, error: userReservationsError, refetch: refetchReservations } =
    useGetUserReservationsQuery();
  const safeUserReservations = userReservations || [];

  const [isAddResourcePopupOpen, setIsAddResourcePopupOpen] = useState(false);
  const [createResource] = useCreateResourceMutation();
  const [setMaintenanceMode] = useSetMaintenanceModeMutation();

  const handleAddResource = async (
    name: string,
    type: string,
    labels: string[],
    properties: { [key: string]: string },
    spaceId?: string,
  ) => {
    const payload = {
      name: name,
      type: type,
      labels: labels,
      properties: properties,
      spaceId: spaceId,
    };
    try {
      await createResource({
        ...payload,
      }).unwrap();
      showNotification("success", t("common:status.success"));
    } catch (e) {
      const error = e as { data?: { error?: string }; message?: string };
      console.error("error creating resource,", error);
      showNotification(
        "error",
        `Error creating resource: ${error?.data?.error || error.message || "Unknown error"}`,
      );
    }
  };

  const filteredResources = useMemo(() => {
    if (!selectedSpaceId) {
      return resourcesWithStatus;
    }
    return resourcesWithStatus.filter(
      (r) => r.resource.spaceId === selectedSpaceId,
    );
  }, [resourcesWithStatus, selectedSpaceId]);

  useEffect(() => {
    if (spaces && spaces.length > 0 && !spaces.some((space) => space.id === selectedSpaceId)) {
      const defaultSpace = spaces.find((s) => s.name === "Default Space");
      if (defaultSpace) setSelectedSpaceId(defaultSpace.id);
      else setSelectedSpaceId(spaces[0].id);
    }
  }, [spaces, selectedSpaceId, setSelectedSpaceId]);

  const handleReserveClick = (resourceId: string) => {
    const resource = resourcesWithStatus.find(
      (r) => r.resource.id === resourceId,
    );
    if (resource) {
      reservationAttempt.current = null;
      setReservationResource({ id: resourceId, name: resource.resource.name });
    }
  };

  const handleConfirmReservation = async (options: ReservationOptions) => {
    if (!reservationResource) return;
    const request = { resourceId: reservationResource.id, ...options };
    const fingerprint = JSON.stringify(request);
    if (reservationAttempt.current?.fingerprint !== fingerprint) {
      reservationAttempt.current = { fingerprint, key: crypto.randomUUID() };
    }
    const reservation = await claimReservation({
      request, idempotencyKey: reservationAttempt.current.key,
    }).unwrap();
    refetchResources();
    showNotification("success", t(reservation.status === "pending"
      ? "components:resourceList.queuedSuccess" : "components:resourceList.reservedSuccess", {
      position: reservation.queuePosition,
    }));
  };

  const handleRelease = async (resourceId: string) => {
    // Find the user's reservation (active or pending) for this resource
    const userReservation = safeUserReservations.find(
      (res: {
        id: string;
        resourceId: string;
        status: string;
        queuePosition: number;
      }) =>
        res.resourceId === resourceId &&
        (res.status === "active" || res.status === "pending"),
    );

    if (userReservation && startAction(resourceId)) {
      try {
        if (userReservation.status === "active") {
          // Complete active reservation
          await completeReservation(userReservation.id).unwrap();
        } else if (userReservation.status === "pending") {
          // Cancel pending reservation (remove from queue)
          await cancelReservation(userReservation.id).unwrap();
        }
        refetchResources();
        showNotification("success", t("common:status.success"));
      } catch (e) {
        const err = e as { data?: { error?: string }; message?: string };
        console.error("Error releasing reservation:", err);
        showNotification(
          "error",
          t("components:resourceList.releaseError", { detail: err?.data?.error || err.message || t("common:errors.generic") }),
        );
      } finally {
        finishAction(resourceId);
      }
    }
  };

  const handleView = (resourceId: string) => {
    const resourceWithStatus = resourcesWithStatus.find(
      (resource: ResourceWithStatus) => resource.resource.id === resourceId,
    );
    if (resourceWithStatus) {
      setViewingResource(resourceWithStatus.resource);
    }
  };

  const handleDelete = async (resourceId: string) => {
    if (!startAction(resourceId)) return;
    try {
      await deleteResource(resourceId).unwrap();
      showNotification("success", t("common:status.success"));
    } catch (e) {
      const err = e as { data?: { error?: string }; message?: string };
      console.error("Error deleting the resource", err);
      showNotification(
        "error",
        `Error deleting resource: ${err?.data?.error || err.message || "Unknown error"}`,
      );
      return;
    } finally {
      finishAction(resourceId);
    }
  };

  const handleCloseModal = () => {
    setViewingResource(null);
  };

  const handleCancelAll = (resourceId: string) => {
    setCancelAllResourceId(resourceId);
  };

  const confirmCancelAll = async () => {
    if (cancelAllResourceId) {
      try {
        await cancelAllReservations(cancelAllResourceId).unwrap();
        setCancelAllResourceId(null);
        refetchResources();
        showNotification("success", t("common:status.success"));
      } catch (e) {
        const err = e as { data?: { error?: string }; message?: string };
        console.error("Error cancelling all reservations:", err);
        showNotification(
          "error",
          `Error cancelling reservations: ${err?.data?.error || err.message || "Unknown error"}`,
        );
      }
    }
  };

  const handleMaintenanceToggle = (
    resourceId: string,
    resourceName: string,
    currentState: boolean,
  ) => {
    setMaintenanceModalResource({
      id: resourceId,
      name: resourceName,
      currentState,
    });
    setMaintenanceReason("");
  };

  const handleConfirmMaintenanceToggle = async () => {
    if (!maintenanceModalResource) return;

    try {
      await setMaintenanceMode({
        resourceId: maintenanceModalResource.id,
        isUnderMaintenance: !maintenanceModalResource.currentState,
        reason: maintenanceReason || undefined,
      }).unwrap();
      setMaintenanceModalResource(null);
      setMaintenanceReason("");
      refetchResources();
      showNotification("success", t("common:status.success"));
    } catch (e) {
      const err = e as { data?: { error?: string }; message?: string };
      console.error("Error toggling maintenance mode:", err);
      showNotification(
        "error",
        `Error toggling maintenance: ${err?.data?.error || err.message || "Unknown error"}`,
      );
    }
  };

  return (
    <div className="flex bg-slate-950 min-h-screen overflow-x-hidden">
      {/* Spaces Sidebar */}
      <div className="w-64 border-r border-slate-800/50 p-4 shrink-0 hidden md:block">
        <h3 className="text-slate-400 font-medium mb-4 flex items-center gap-2">
          <Box className="w-4 h-4" />
          {t("pages:home.spaces")}
        </h3>
        <div className="space-y-1">
          {spaces?.map((space: Space) => (
            <button
              key={space.id}
              onClick={() => setSelectedSpaceId(space.id)}
              className={`w-full text-left px-3 py-2 rounded-lg text-sm transition-colors ${
                selectedSpaceId === space.id
                  ? "bg-cyan-600/10 text-cyan-400 border border-cyan-600/20"
                  : "text-slate-400 hover:text-slate-200 hover:bg-slate-900"
              }`}
            >
              {space.name}
            </button>
          ))}
        </div>
      </div>

      <div className="p-4 sm:p-6 w-full min-w-0">
      {spaces && spaces.length > 0 && (
        <div className="md:hidden mb-6">
          <label htmlFor="mobile-space" className="block text-sm font-medium text-slate-400 mb-2">
            {t("pages:home.spaces")}
          </label>
          <select
            id="mobile-space"
            value={selectedSpaceId || ""}
            onChange={(e) => setSelectedSpaceId(e.target.value)}
            className="w-full bg-slate-800/50 border border-slate-700/50 rounded-lg px-4 py-3 text-slate-200 focus:outline-none focus:border-cyan-500 transition-colors"
          >
            {spaces.map((space: Space) => (
              <option key={space.id} value={space.id}>
                {space.name}
              </option>
            ))}
          </select>
        </div>
      )}

        {apiError && (
          <div role="alert" className="glass-panel p-4 rounded-xl border-brand-busy/20 bg-brand-busy/5 mb-6 text-center">
            <p className="text-brand-busy font-medium">
              {t("pages:home.fetchError")}
            </p>
            <button type="button" onClick={() => refetchResources()} className="underline p-2">{t("common:refresh")}</button>
          </div>
        )}

        {userReservationsError && <div role="alert" className="text-red-400 mb-4">
          <p>{t("pages:home.reservationsError")}</p>
          <button type="button" onClick={() => refetchReservations()} className="underline p-2">{t("common:refresh")}</button>
        </div>}

        <div className="flex flex-wrap gap-4 items-center justify-between mb-6">
          <div>
            <h2 className="text-2xl font-bold text-white tracking-tight">
              {t("pages:home.title")}
            </h2>
            <p className="text-slate-400 text-sm">{t("pages:home.subtitle")}</p>
          </div>

          {isAdmin && (
            <button
              onClick={() => setIsAddResourcePopupOpen(true)}
              className="btn-primary"
            >
              <Plus className="w-4 h-4" />
              {t("pages:home.addResource")}
            </button>
          )}
        </div>

        {(isFetching || isUserReservationsFetching) && !isLoading && (
          <p role="status" className="text-sm text-slate-400 mb-3">{t("pages:home.refreshingResources")}</p>
        )}
        {isLoading && resourcesWithStatus.length === 0 ? (
          <div role="status" className="flex flex-col items-center justify-center p-20 glass-panel rounded-xl">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-brand-queued mb-4"></div>
            <p className="text-slate-400 font-medium">
              {t("pages:home.loadingResources")}
            </p>
          </div>
        ) : (
          <ResourceTable
            data={filteredResources}
            userReservations={safeUserReservations}
            onReserve={handleReserveClick}
            onRelease={handleRelease}
            onView={handleView}
            onDelete={handleDelete}
            onCancelAll={handleCancelAll}
            onMaintenanceToggle={isAdmin ? handleMaintenanceToggle : undefined}
            isAdmin={isAdmin}
            busyResourceIds={busyResourceIds}
            reservationsReady={!isUserReservationsLoading && !isUserReservationsFetching && !userReservationsError}
          />
        )}

        {viewingResource && (
          <ResourceDetailsModal
            resource={viewingResource}
            onClose={handleCloseModal}
          />
        )}

        <AddResourcePopup
          isOpen={isAddResourcePopupOpen}
          onClose={() => setIsAddResourcePopupOpen(false)}
          onSave={(name, type, labels, properties, spaceId) => {
            handleAddResource(name, type, labels, properties, spaceId);
            setIsAddResourcePopupOpen(false);
          }}
          preSelectedSpaceId={selectedSpaceId || undefined}
        />

        <ConfirmationModal
          isOpen={!!cancelAllResourceId}
          onClose={() => setCancelAllResourceId(null)}
          onConfirm={confirmCancelAll}
          title={t("components:confirmationModal.cancelAllTitle")}
          message={t("components:confirmationModal.cancelAllMessage")}
          confirmText={t("components:confirmationModal.confirmYes")}
          cancelText={t("components:confirmationModal.confirmNo")}
          isDestructive={true}
        />

        {reservationResource && (
          <ReservationModal
            isOpen={!!reservationResource}
            onClose={() => setReservationResource(null)}
            resourceName={reservationResource.name}
            onConfirm={handleConfirmReservation}
          />
        )}

        {maintenanceModalResource && (
          <ConfirmationModal
            isOpen={!!maintenanceModalResource}
            onClose={() => {
              setMaintenanceModalResource(null);
              setMaintenanceReason("");
            }}
            onConfirm={handleConfirmMaintenanceToggle}
            title={
              maintenanceModalResource.currentState
                ? t("components:resourceList.maintenance.disable")
                : t("components:resourceList.maintenance.enable")
            }
            message={
              <div>
                <p className="mb-4">
                  {maintenanceModalResource.currentState
                    ? t("components:resourceList.maintenance.confirmDisable", {
                        name: maintenanceModalResource.name,
                      })
                    : t("components:resourceList.maintenance.confirmEnable", {
                        name: maintenanceModalResource.name,
                      })}
                </p>
                {!maintenanceModalResource.currentState && (
                  <div>
                    <label className="block text-sm text-slate-400 mb-2">
                      {t("components:resourceList.maintenance.reason")}
                    </label>
                    <textarea
                      value={maintenanceReason}
                      onChange={(e) => setMaintenanceReason(e.target.value)}
                      placeholder={t(
                        "components:resourceList.maintenance.reasonPlaceholder",
                      )}
                      className="w-full rounded-lg bg-slate-800 border border-slate-700 p-3 text-sm text-slate-200 focus:outline-none focus:border-cyan-500"
                      rows={3}
                    />
                  </div>
                )}
              </div>
            }
            confirmText={
              maintenanceModalResource.currentState
                ? t("components:resourceList.maintenance.disable")
                : t("components:resourceList.maintenance.enable")
            }
            cancelText={t("common:cancel")}
            isDestructive={!maintenanceModalResource.currentState}
          />
        )}
      </div>
    </div>
  );
};
export default ResourceList;
