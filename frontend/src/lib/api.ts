const API_BASE = "http://localhost:8080/api/v1"

export interface SessionData {
    token: string
    user_id: string
    user_name: string
    user_email: string
    user_type: string
}

export interface Property {
    id: string
    owner_id: number
    title: string
    address: string
    created_at: string
    updated_at: string
    status: "occupied" | "vacant"
    guest_name?: string
    current_check_in?: string
    current_check_out?: string
    next_check_in?: string
}

export interface Reservation {
    id: number
    property_id: string
    property_name: string
    booked_by: number
    guest_name: string
    check_in: string
    check_out: string
    created_at: string
    updated_at: string
}

export interface PaginatedResponse<T> {
    properties?: T[]
    reservations?: T[]
    total: number
    page: number
    pages: number
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
    const session = JSON.parse(localStorage.getItem("rented_session") || "null")
    const headers: Record<string, string> = {
        "Content-Type": "application/json",
        ...(options.headers as Record<string, string> || {}),
    }

    if (session?.token) {
        headers["Authorization"] = `Bearer ${session.token}`
    }

    const res = await fetch(`${API_BASE}${path}`, {
        ...options,
        headers,
    })

    let data: unknown = null
    try {
        data = await res.json()
    } catch {
        // Some failures can return an empty or non-JSON body.
    }

    if (!res.ok) {
        const message = data && typeof data === "object" && "error" in data && typeof data.error === "string"
            ? data.error
            : "request failed"

        if (res.status === 401) {
            localStorage.removeItem("rented_session")
        }

        throw new Error(message)
    }

    return data as T
}

function withQuery(path: string, params: Record<string, string | number | undefined>): string {
    const query = new URLSearchParams()

    for (const [key, value] of Object.entries(params)) {
        if (value !== undefined && value !== "") {
            query.set(key, String(value))
        }
    }

    const queryString = query.toString()
    return queryString ? `${path}?${queryString}` : path
}

export const api = {
    signup: (name: string, email: string, password: string, type: string) =>
        request<SessionData>("/signup", {
            method: "POST",
            body: JSON.stringify({ name, email, password, type }),
        }),

    login: (email: string, password: string) =>
        request<SessionData>("/login", {
            method: "POST",
            body: JSON.stringify({ email, password }),
        }),

    logout: () => request("/logout", { method: "POST" }),

    listProperties: (page: number = 1) =>
        request<PaginatedResponse<Property>>(withQuery("/property", { page }), { method: "GET" }),

    createProperty: (title: string, address: string) =>
        request<Property>("/property/new", {
            method: "PUT",
            body: JSON.stringify({ title, address }),
        }),

    listReservations: (page: number = 1, propertyName?: string, guestName?: string, checkInFrom?: string, checkOutTo?: string) =>
        request<PaginatedResponse<Reservation>>(withQuery("/reservation", {
            page,
            property_name: propertyName,
            guest_name: guestName,
            check_in_from: checkInFrom,
            check_out_to: checkOutTo,
        }), { method: "GET" }),

    createReservation: (propertyId: string, guestName: string, checkIn: string, checkOut: string) =>
        request<Reservation>("/reservation/new", {
            method: "PUT",
            body: JSON.stringify({
                property_id: propertyId,
                guest_name: guestName,
                checkin: checkIn,
                checkout: checkOut,
                timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
            }),
        }),

    updateReservation: (id: number, propertyId: string, guestName: string, checkIn: string, checkOut: string) =>
        request<Reservation>(`/reservation/edit/${id}`, {
            method: "PATCH",
            body: JSON.stringify({
                property_id: propertyId,
                guest_name: guestName,
                checkin: checkIn,
                checkout: checkOut,
                timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
            }),
        }),
}
