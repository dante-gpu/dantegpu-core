import { Link } from "react-router-dom";
import { HardDrive, Plus } from "lucide-react";
import { useRentals } from "@/hooks/useRentals";
import { RentalRow } from "@/components/RentalRow";
import { Card, CardBody, CardHeader } from "@/components/ui/Card";
import { Button } from "@/components/ui/Button";
import { EmptyState } from "@/components/ui/EmptyState";

export default function MyRentals() {
  const rentals = useRentals();

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-ink-50">My Rentals</h1>
          <p className="mt-1 text-sm text-ink-400">Sessions you have started, newest first.</p>
        </div>
        <Link to="/marketplace">
          <Button>
            <Plus className="size-4" /> New rental
          </Button>
        </Link>
      </div>

      {rentals.length === 0 ? (
        <EmptyState
          icon={<HardDrive className="size-8" />}
          title="You have not rented any GPUs yet"
          description="Pick a GPU from the marketplace to start your first metered session."
          action={
            <Link to="/marketplace">
              <Button>Browse marketplace</Button>
            </Link>
          }
        />
      ) : (
        <Card>
          <CardHeader title={`${rentals.length} ${rentals.length === 1 ? "rental" : "rentals"}`} />
          <CardBody className="pt-1">
            <div className="divide-y divide-ink-700">
              {rentals.map((r) => (
                <RentalRow key={r.jobId} rental={r} />
              ))}
            </div>
          </CardBody>
        </Card>
      )}
    </div>
  );
}
