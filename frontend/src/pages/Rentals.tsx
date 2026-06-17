import { useState, useEffect, type FormEvent } from "react"
import { KeyRound, Plus } from "lucide-react"
import { api, type Reservation, type Property } from "@/lib/api"
import { errorMessage } from "@/lib/error-message"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Field, FieldGroup, FieldLabel, FieldError } from "@/components/ui/field"
import { Spinner } from "@/components/ui/spinner"
import { useDebouncedValue } from "@/hooks/useDebouncedValue"
import {
    Select,
    SelectContent,
    SelectGroup,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select"
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
    DialogTrigger,
} from "@/components/ui/dialog"
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table"
import { PaginationBar } from "@/components/ui/pagination-bar"
import { EmptyState } from "@/components/ui/empty-state"

function formatDateTime(dt: string) {
    return new Date(dt).toLocaleString(undefined, {
        dateStyle: "medium",
        timeStyle: "short",
    })
}

const PAGE_SIZE = 50

export default function Rentals() {
    const [reservations, setReservations] = useState<Reservation[]>([])
    const [page, setPage] = useState(1)
    const [pages, setPages] = useState(1)
    const [loading, setLoading] = useState(true)
    const [dialogOpen, setDialogOpen] = useState(false)
    const [detailDialogOpen, setDetailDialogOpen] = useState(false)
    const [selectedReservation, setSelectedReservation] = useState<Reservation | null>(null)
    const [properties, setProperties] = useState<Property[]>([])
    const [propertyId, setPropertyId] = useState("")
    const [guestName, setGuestName] = useState("")
    const [checkIn, setCheckIn] = useState(() => {
        const d = new Date()
        d.setHours(11, 0, 0, 0)
        return d.toISOString().slice(0, 16)
    })
    const [checkOut, setCheckOut] = useState(() => {
        const d = new Date()
        d.setDate(d.getDate() + 1)
        d.setHours(10, 30, 0, 0)
        return d.toISOString().slice(0, 16)
    })
    const [error, setError] = useState("")
    const [loadError, setLoadError] = useState("")
    const [creating, setCreating] = useState(false)

    const [filterProperty, setFilterProperty] = useState("")
    const [filterGuest, setFilterGuest] = useState("")
    const [filterCheckInFrom, setFilterCheckInFrom] = useState("")
    const [filterCheckOutTo, setFilterCheckOutTo] = useState("")

    const debouncedProperty = useDebouncedValue(filterProperty, 300)
    const debouncedGuest = useDebouncedValue(filterGuest, 300)

    const hasFilters = filterProperty !== "" || filterGuest !== "" || filterCheckInFrom !== "" || filterCheckOutTo !== ""

    useEffect(() => {
        let cancelled = false
        setLoading(true)
        setLoadError("")
        const checkInFrom = filterCheckInFrom ? new Date(filterCheckInFrom).toISOString() : undefined
        const checkOutTo = filterCheckOutTo ? new Date(filterCheckOutTo).toISOString() : undefined
        api.listReservations(page, debouncedProperty, debouncedGuest, checkInFrom, checkOutTo).then((data) => {
            if (!cancelled) {
                setReservations(data.reservations || [])
                setPages(data.pages)
            }
        }).catch((err) => {
            if (!cancelled) {
                setLoadError(errorMessage(err, "failed to load reservations"))
            }
        }).finally(() => {
            if (!cancelled) setLoading(false)
        })
        return () => { cancelled = true }
    }, [page, debouncedProperty, debouncedGuest, filterCheckInFrom, filterCheckOutTo])

    useEffect(() => {
        if (page !== 1) setPage(1)
    }, [debouncedProperty, debouncedGuest, filterCheckInFrom, filterCheckOutTo])

    useEffect(() => {
        if (dialogOpen) {
            api.listProperties(1).then((data) => {
                setProperties(data.properties || [])
            }).catch((err) => {
                setError(errorMessage(err, "failed to load properties"))
            })
        }
    }, [dialogOpen])

    async function handleCreate(e: FormEvent) {
        e.preventDefault()
        setError("")

        if (new Date(checkIn) < new Date()) {
            setError("Check-in cannot be in the past")
            return
        }

        if (new Date(checkOut) <= new Date(checkIn)) {
            setError("Check-out must be after check-in")
            return
        }

        setCreating(true)
        try {
            const newReservation = await api.createReservation(propertyId, guestName, checkIn, checkOut)
            setDialogOpen(false)
            setPropertyId("")
            setGuestName("")
            const d = new Date()
            d.setHours(11, 0, 0, 0)
            setCheckIn(d.toISOString().slice(0, 16))
            const d2 = new Date()
            d2.setDate(d2.getDate() + 1)
            d2.setHours(10, 30, 0, 0)
            setCheckOut(d2.toISOString().slice(0, 16))

            if (page === 1) {
                setReservations([newReservation, ...reservations])
                setPages(Math.ceil((reservations.length + 1) / PAGE_SIZE))
            } else {
                setPage(1)
            }
        } catch (err) {
            setError(errorMessage(err, "failed to create reservation"))
        } finally {
            setCreating(false)
        }
    }

    function handleRowClick(reservation: Reservation) {
        setSelectedReservation(reservation)
        setDetailDialogOpen(true)
    }

    return (
        <div className="flex flex-col gap-6 h-full min-h-0">
            <div className="flex flex-col gap-4 shrink-0">
                <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
                    <div>
                        <h1 className="text-2xl font-semibold">Reservations</h1>
                        <p className="text-muted-foreground">Manage your reservations.</p>
                    </div>
                    <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
                        <DialogTrigger render={<Button />}>
                            <Plus data-icon="inline-start" />
                            New
                        </DialogTrigger>
                        <DialogContent>
                            <DialogHeader>
                                <DialogTitle>Create Reservation</DialogTitle>
                                <DialogDescription>Book a property for a guest.</DialogDescription>
                            </DialogHeader>
                            <form onSubmit={handleCreate}>
                                <FieldGroup>
                                    <Field>
                                        <FieldLabel htmlFor="property">Property</FieldLabel>
                                        <Select value={propertyId} onValueChange={(val) => setPropertyId(val || "")}>
                                            <SelectTrigger className="w-full">
                                                <SelectValue placeholder="Select a property" />
                                            </SelectTrigger>
                                            <SelectContent>
                                                <SelectGroup>
                                                    {properties.map((p) => (
                                                        <SelectItem key={p.id} value={p.id}>
                                                            {p.title}
                                                        </SelectItem>
                                                    ))}
                                                </SelectGroup>
                                            </SelectContent>
                                        </Select>
                                    </Field>
                                    <Field>
                                        <FieldLabel htmlFor="guestname">Guest Name</FieldLabel>
                                        <Input
                                            id="guestname"
                                            value={guestName}
                                            onChange={(e) => setGuestName(e.target.value)}
                                            placeholder="Guest name"
                                            required
                                        />
                                    </Field>
                                    <Field>
                                        <FieldLabel htmlFor="checkin">Check-in</FieldLabel>
                                        <Input
                                            id="checkin"
                                            type="datetime-local"
                                            value={checkIn}
                                            onChange={(e) => setCheckIn(e.target.value)}
                                            required
                                        />
                                    </Field>
                                    <Field>
                                        <FieldLabel htmlFor="checkout">Check-out</FieldLabel>
                                        <Input
                                            id="checkout"
                                            type="datetime-local"
                                            value={checkOut}
                                            onChange={(e) => setCheckOut(e.target.value)}
                                            required
                                        />
                                    </Field>
                                    {error && <FieldError>{error}</FieldError>}
                                </FieldGroup>
                                <DialogFooter>
                                    <Button type="submit" disabled={creating}>
                                        {creating && <Spinner data-icon="inline-start" />}
                                        Create
                                    </Button>
                                </DialogFooter>
                            </form>
                        </DialogContent>
                    </Dialog>
                </div>
                <div className="flex flex-wrap items-center gap-2">
                    <Input
                        placeholder="Property"
                        value={filterProperty}
                        onChange={(e) => setFilterProperty(e.target.value)}
                        className="w-full sm:w-32"
                    />
                    <Input
                        placeholder="Guest"
                        value={filterGuest}
                        onChange={(e) => setFilterGuest(e.target.value)}
                        className="w-full sm:w-32"
                    />
                    <Input
                        type="datetime-local"
                        value={filterCheckInFrom}
                        onChange={(e) => setFilterCheckInFrom(e.target.value)}
                        className="w-full sm:w-48"
                    />
                    <Input
                        type="datetime-local"
                        value={filterCheckOutTo}
                        onChange={(e) => setFilterCheckOutTo(e.target.value)}
                        className="w-full sm:w-48"
                    />
                </div>
            </div>

            <div className="rounded-lg border bg-card flex flex-col flex-1 overflow-hidden min-h-0 max-h-[50vh] md:max-h-[70vh]">
                {loadError ? (
                    <div className="flex-1 flex items-center justify-center p-6 text-sm text-destructive">
                        {loadError}
                    </div>
                ) : loading ? (
                    <div className="flex-1 flex items-center justify-center p-12">
                        <Spinner />
                    </div>
                ) : (reservations?.length ?? 0) === 0 && !hasFilters ? (
                    <div className="flex-1 flex items-center justify-center">
                        <EmptyState
                            icon={KeyRound}
                            title="No reservations yet"
                            description="Reservations you create will appear here."
                            className="border-0"
                        />
                    </div>
                ) : (reservations?.length ?? 0) === 0 && hasFilters ? (
                    <div className="flex-1 flex items-center justify-center">
                        <EmptyState
                            icon={KeyRound}
                            title="No matching reservations"
                            description="Try adjusting your filters."
                            className="border-0"
                        />
                    </div>
                ) : (
                    <div className="flex-1 overflow-y-auto min-h-0 max-full">
                        <Table className="[&_thead]:sticky [&_thead]:top-0 [&_thead]:z-10 [&_thead]:bg-card">
                            <TableHeader>
                                <TableRow>
                                    <TableHead>Property</TableHead>
                                    <TableHead>Guest</TableHead>
                                    <TableHead>Check-in</TableHead>
                                    <TableHead>Check-out</TableHead>
                                </TableRow>
                            </TableHeader>
                            <TableBody>
                                {reservations.map((res) => (
                                    <TableRow
                                        key={res.id}
                                        className="cursor-pointer"
                                        onClick={() => handleRowClick(res)}
                                    >
                                        <TableCell className="font-medium">{res.property_name}</TableCell>
                                        <TableCell>{res.guest_name}</TableCell>
                                        <TableCell>{formatDateTime(res.check_in)}</TableCell>
                                        <TableCell>{formatDateTime(res.check_out)}</TableCell>
                                    </TableRow>
                                ))}
                            </TableBody>
                        </Table>
                    </div>
                )}
            </div>

            <div className="shrink-0">
                <PaginationBar page={page} pages={pages} onPageChange={setPage} />
            </div>

            <Dialog open={detailDialogOpen} onOpenChange={setDetailDialogOpen}>
                <DialogContent className="sm:max-w-md">
                    <DialogHeader>
                        <DialogTitle>Reservation Details</DialogTitle>
                        <DialogDescription>
                            Complete information about this reservation
                        </DialogDescription>
                    </DialogHeader>
                    {selectedReservation && (
                        <div className="grid gap-4 py-4">
                            <div className="grid gap-2">
                                <div className="text-sm font-medium text-muted-foreground">Property</div>
                                <div className="text-lg font-semibold">{selectedReservation.property_name}</div>
                            </div>
                            <div className="grid gap-2">
                                <div className="text-sm font-medium text-muted-foreground">Guest Name</div>
                                <div>{selectedReservation.guest_name}</div>
                            </div>
                            <div className="grid gap-2">
                                <div className="text-sm font-medium text-muted-foreground">Check-in</div>
                                <div>{formatDateTime(selectedReservation.check_in)}</div>
                            </div>
                            <div className="grid gap-2">
                                <div className="text-sm font-medium text-muted-foreground">Check-out</div>
                                <div>{formatDateTime(selectedReservation.check_out)}</div>
                            </div>
                            <div className="grid gap-2">
                                <div className="text-sm font-medium text-muted-foreground">Created</div>
                                <div className="text-sm">{new Date(selectedReservation.created_at).toLocaleString()}</div>
                            </div>
                        </div>
                    )}
                </DialogContent>
            </Dialog>
        </div>
    )
}
