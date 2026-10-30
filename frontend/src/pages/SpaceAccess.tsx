import { Link, useParams } from "react-router-dom";
import { useGetSpacesQuery } from "../store/api/spaces";
import { SpaceAccessPanel } from "../components/SpaceAccessPanel";

const SpaceAccess = () => {
  const { spaceId } = useParams();
  const { data: spaces = [], isLoading, error, refetch } = useGetSpacesQuery();
  const space = spaces.find((item) => item.id === spaceId);
  return (
    <div className="space-y-6">
      <Link className="text-cyan-300 underline text-sm" to="/admin/spaces">
        Back to spaces
      </Link>
      {isLoading ? (
        <p role="status">Loading space…</p>
      ) : error ? (
        <div role="alert">
          <p>Could not load this space.</p>
          <button className="btn-ghost mt-3" onClick={() => refetch()}>
            Retry
          </button>
        </div>
      ) : !space ? (
        <h1 className="text-2xl text-white">Space not found</h1>
      ) : (
        <>
          <header>
            <p className="text-sm text-slate-400 mb-2">
              Administration / Spaces / Access
            </p>
            <h1 className="text-3xl font-semibold text-white">{space.name}</h1>
            <p className="text-slate-400 mt-2">
              {space.description || "Manage access to this space."}
            </p>
          </header>
          <div className="border-b border-slate-700 pb-3 text-cyan-300 font-medium">
            Access
          </div>
          <SpaceAccessPanel key={space.id} space={space} />
        </>
      )}
    </div>
  );
};

export default SpaceAccess;
