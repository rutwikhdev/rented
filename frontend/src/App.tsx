import { Routes, Route, Navigate } from "react-router-dom"
import { AuthProvider } from "@/contexts/AuthProvider"
import { ProtectedRoute } from "@/components/ProtectedRoute"
import { DashboardLayout } from "@/components/layout/DashboardLayout"
import Login from "@/pages/Login"
import Signup from "@/pages/Signup"
import Properties from "@/pages/Properties"
import Rentals from "@/pages/Rentals"

function App() {
  return (
    <AuthProvider>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="/signup" element={<Signup />} />
        <Route
          path="/"
          element={<Navigate to="/properties" replace />}
        />
        <Route
          path="/properties"
          element={
            <ProtectedRoute>
              <DashboardLayout>
                <Properties />
              </DashboardLayout>
            </ProtectedRoute>
          }
        />
        <Route
          path="/rentals"
          element={
            <ProtectedRoute>
              <DashboardLayout>
                <Rentals />
              </DashboardLayout>
            </ProtectedRoute>
          }
        />
      </Routes>
    </AuthProvider>
  )
}

export default App
