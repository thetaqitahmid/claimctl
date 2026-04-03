# Reservations & Queue System

This guide explains how to book resources and how the queue system manages
demand.

## Making a Reservation

1. **Find an Item**: Browse the resource list to find what you need.
2. **Reserve**:
   - If the item is **Available**: Click "Reserve". You become the active holder
     immediately.
   - If the item is **Reserved**: Click "Join Queue". You will be added to the
     waiting list.

## Reservation Lifecycle

A reservation goes through several states:

1. **Pending (Queue)**: You are waiting for the resource to become available.
2. **Active**: You typically enter this state immediately if the resource was
   free. If you were in the queue, you become Active when the previous user
   releases the item *and* you confirm/activate your turn.
3. **Completed**: You have finished using the resource and returned it.
4. **Cancelled**: You decided you no longer need the resource.

## The Queue System

When a resource is busy, multiple users can queue for it.
- **First-Come-First-Served**: The queue is strictly ordered.
- **Position**: You can see your position (e.g., "3rd in line") on the resource
  card.
- **Next in Line**: When the current user completes their reservation, the next
  person in the queue is notified (or auto-assigned depending on configuration).

## My Reservations

You can view all your current and past activity in the **Profile** or **My
Reservations** section.
- **Active**: Items currently in your possession.
- **Queue**: Items you are waiting for.
- **History**: A log of your past completed reservations.
