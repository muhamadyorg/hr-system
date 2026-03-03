import { Switch, Route } from "wouter";
import { queryClient } from "./lib/queryClient";
import { QueryClientProvider } from "@tanstack/react-query";
import { Toaster } from "@/components/ui/toaster";
import { TooltipProvider } from "@/components/ui/tooltip";
import { AuthProvider, useAuth } from "@/lib/auth";
import NotFound from "@/pages/not-found";
import LoginPage from "@/pages/login";
import DashboardPage from "@/pages/dashboard";
import EmployeesPage from "@/pages/employees";
import EmployeeDetailPage from "@/pages/employee-detail";
import GroupsPage from "@/pages/groups";
import AttendancePage from "@/pages/attendance";
import SudoPage from "@/pages/sudo";
import AdminSettingsPage from "@/pages/admin-settings";
import { useEffect } from "react";
import { Loader2 } from "lucide-react";

function DarkModeEnforcer() {
  useEffect(() => {
    document.documentElement.classList.add("dark");
  }, []);
  return null;
}

function AuthGuard({ children }: { children: React.ReactNode }) {
  const { user, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    );
  }

  if (!user) {
    return <LoginPage />;
  }

  return <>{children}</>;
}

function Router() {
  return (
    <Switch>
      <Route path="/" component={DashboardPage} />
      <Route path="/employees" component={EmployeesPage} />
      <Route path="/employees/:id" component={EmployeeDetailPage} />
      <Route path="/groups" component={GroupsPage} />
      <Route path="/attendance" component={AttendancePage} />
      <Route path="/sudo" component={SudoPage} />
      <Route path="/admin-settings" component={AdminSettingsPage} />
      <Route component={NotFound} />
    </Switch>
  );
}

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <TooltipProvider>
        <AuthProvider>
          <DarkModeEnforcer />
          <AuthGuard>
            <Router />
          </AuthGuard>
          <Toaster />
        </AuthProvider>
      </TooltipProvider>
    </QueryClientProvider>
  );
}

export default App;
