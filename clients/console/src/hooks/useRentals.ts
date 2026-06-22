import { useSyncExternalStore } from "react";
import { rentalsStore, type LocalRental } from "@/lib/rentalsStore";

// Subscribes to the local rentals store. Components re-render whenever a rental
// is added or removed.
export function useRentals(): LocalRental[] {
  return useSyncExternalStore(rentalsStore.subscribe, rentalsStore.getSnapshot, rentalsStore.getSnapshot);
}
