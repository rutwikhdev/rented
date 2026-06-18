import { useState, useEffect, useRef, type FormEvent } from "react"
import { Building2, Plus } from "lucide-react"
import { useAuth } from "@/hooks/useAuth"
import { api, type Property } from "@/lib/api"
import { errorMessage } from "@/lib/error-message"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldGroup, FieldLabel, FieldError } from "@/components/ui/field"
import { Spinner } from "@/components/ui/spinner"
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
    DialogTrigger,
} from "@/components/ui/dialog"
import { PaginationBar } from "@/components/ui/pagination-bar"
import { EmptyState } from "@/components/ui/empty-state"

const PAGE_SIZE = 50

function formatDateTime(dt: string) {
    return new Date(dt).toLocaleString(undefined, {
        dateStyle: "medium",
        timeStyle: "short",
    })
}

export default function Properties() {
    const { user } = useAuth()
    const [properties, setProperties] = useState<Property[]>([])
    const [page, setPage] = useState(1)
    const [pages, setPages] = useState(1)
    const [loading, setLoading] = useState(true)
    const [dialogOpen, setDialogOpen] = useState(false)
    const [title, setTitle] = useState("")
    const [address, setAddress] = useState("")
    const [error, setError] = useState("")
    const [loadError, setLoadError] = useState("")
    const [creating, setCreating] = useState(false)

    const prevPage = useRef(page)

    useEffect(() => {
        if (prevPage.current !== page || properties.length === 0) {
            prevPage.current = page
            let cancelled = false
            setLoading(true)
            setLoadError("")
            api.listProperties(page).then((data) => {
                if (!cancelled) {
                    setProperties(data.properties || [])
                    setPages(data.pages)
                }
            }).catch((err) => {
                if (!cancelled) {
                    setLoadError(errorMessage(err, "failed to load properties"))
                }
            }).finally(() => {
                if (!cancelled) setLoading(false)
            })
            return () => { cancelled = true }
        }
    }, [page])

    async function handleCreate(e: FormEvent) {
        e.preventDefault()
        setError("")
        setCreating(true)
        try {
            const newProperty = await api.createProperty(title, address)
            setDialogOpen(false)
            setTitle("")
            setAddress("")

            if (page === 1) {
                setProperties([newProperty, ...properties])
                setPages(Math.ceil((properties.length + 1) / PAGE_SIZE))
            } else {
                prevPage.current = -1
                setPage(1)
            }
        } catch (err) {
            setError(errorMessage(err, "failed to create property"))
        } finally {
            setCreating(false)
        }
    }

    return (
        <div className="flex flex-col gap-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-semibold">Properties</h1>
                    <p className="text-muted-foreground">Manage your properties.</p>
                </div>
                {user?.type === "manager" && (
                    <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
                        <DialogTrigger render={<Button />}>
                            <Plus data-icon="inline-start" />
                            New Property
                        </DialogTrigger>
                        <DialogContent>
                            <DialogHeader>
                                <DialogTitle>Create Property</DialogTitle>
                                <DialogDescription>Add a new property to your portfolio.</DialogDescription>
                            </DialogHeader>
                            <form onSubmit={handleCreate}>
                                <FieldGroup>
                                    <Field>
                                        <FieldLabel htmlFor="title">Title</FieldLabel>
                                        <Input
                                            id="title"
                                            value={title}
                                            onChange={(e) => setTitle(e.target.value)}
                                            placeholder="Property title"
                                            required
                                        />
                                    </Field>
                                    <Field>
                                        <FieldLabel htmlFor="address">Address</FieldLabel>
                                        <Input
                                            id="address"
                                            value={address}
                                            onChange={(e) => setAddress(e.target.value)}
                                            placeholder="Full address"
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
                )}
            </div>

            {loadError ? (
                <div className="rounded-lg border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive">
                    {loadError}
                </div>
            ) : loading ? (
                <div className="flex justify-center p-12">
                    <Spinner />
                </div>
            ) : properties.length === 0 ? (
                <EmptyState
                    icon={Building2}
                    title="No properties yet"
                    description="Properties you add will appear here."
                />
            ) : (
                <>
                    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
                        {properties.map((property) => (
                            <Card key={property.id}>
                                <CardHeader>
                                    <CardTitle>{property.title}</CardTitle>
                                </CardHeader>
                                <CardContent className="flex flex-col gap-3">
                                    <p className="text-sm text-muted-foreground">{property.address}</p>
                                    <div className="flex flex-col gap-1 rounded-md border p-3 text-sm">
                                        <div className="flex">
                                            <p className="pr-2">Status: </p>
                                            <div
                                                className={cn(
                                                    "inline-flex w-fit rounded-full border px-2 py-0.5 text-xs font-medium capitalize",
                                                    property.status === "occupied"
                                                        ? "border-yellow-500/30 bg-yellow-500/10 text-yellow-700"
                                                        : "border-green-500/30 bg-green-500/10 text-green-700",
                                                )}
                                            >
                                                {property.status}
                                            </div>
                                        </div>
                                        {property.status === "occupied" && property.guest_name && (
                                            <div className="text-muted-foreground">
                                                Current guest: {property.guest_name}
                                            </div>
                                        )}
                                        {property.status === "vacant" && property.guest_name && (
                                            <div className="text-muted-foreground">
                                                Next guest: {property.guest_name}
                                            </div>
                                        )}
                                        {property.next_check_in ? (
                                            <div className="text-muted-foreground">
                                                Next check-in: {formatDateTime(property.next_check_in)}
                                            </div>
                                        ) : (
                                            <div className="text-muted-foreground">No upcoming check-in</div>
                                        )}
                                    </div>
                                </CardContent>
                            </Card>
                        ))}
                    </div>
                    <PaginationBar page={page} pages={pages} onPageChange={setPage} />
                </>
            )}
        </div>
    )
}
