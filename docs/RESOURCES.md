# Resource Management

Resources are the core assets in claimctl that users can view and book.

## Overview

A **Resource** can be anything: a meeting room, a laptop, a microscope, or a
library book. Each resource belongs to a **Space** (a logical grouping like "Lab
A" or "IT Dept").

## Managing Resources (Admins)

Admins can manage resources via the Admin Panel.

### Creating a Resource

To add a new resource:

1. Navigate to the Admin Panel.
2. Click **Create Resource**.
3. Fill in the details:
   - **Name**: Unique name for the item.
   - **Type**: Category (e.g., "Equipment", "Room").
   - **Status**: Initial status (Available/Maintenance).
   - **Space**: The space this resource belongs to.

### Properties

Resources can have arbitrary properties (key-value pairs) for detailed tracking
(e.g., `Serial Number: 12345`, `Color: Blue`).

## Viewing Resources

Users can browse resources on the main dashboard.

- **Filtering**: Filter by Space or Type.
- **Status Indicators**:
  - `Available`: Ready to be booked.
  - `Reserved`: Currently in use.
  - `Maintenance`: Temporarily unavailable.

## Resource History

Clicking on a resource allows you to view its **History** (if you have
permission).

- Shows a timeline of who reserved the item and when.
- Useful for tracking usage patterns or locating missing items.
