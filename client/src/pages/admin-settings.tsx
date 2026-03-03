import { useLocation } from "wouter";
import { useAuth } from "@/lib/auth";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { ArrowLeft, ShieldCheck, Users, FolderOpen, CalendarCheck } from "lucide-react";

export default function AdminSettingsPage() {
  const [, navigate] = useLocation();
  const { user } = useAuth();

  return (
    <div className="min-h-screen bg-background">
      <header className="border-b border-border sticky top-0 z-50 bg-background">
        <div className="max-w-4xl mx-auto px-4 py-3 flex items-center gap-2">
          <Button variant="ghost" size="icon" onClick={() => navigate("/")} data-testid="button-back">
            <ArrowLeft className="w-4 h-4" />
          </Button>
          <div className="flex items-center gap-2">
            <ShieldCheck className="w-5 h-5 text-primary" />
            <h1 className="text-lg font-bold text-foreground">Admin Panel</h1>
          </div>
        </div>
      </header>

      <main className="max-w-4xl mx-auto px-4 py-6 space-y-4">
        <Card>
          <CardContent className="p-5">
            <div className="flex items-center gap-3 mb-4">
              <div className="w-12 h-12 rounded-md bg-primary/10 flex items-center justify-center">
                <ShieldCheck className="w-6 h-6 text-primary" />
              </div>
              <div>
                <h2 className="text-lg font-semibold text-card-foreground">{user?.fullName}</h2>
                <p className="text-sm text-muted-foreground">@{user?.username}</p>
              </div>
              <Badge variant="secondary" className="ml-auto">ADMIN</Badge>
            </div>
          </CardContent>
        </Card>

        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
          <button onClick={() => navigate("/employees")} className="focus:outline-none focus:ring-2 focus:ring-ring rounded-md">
            <Card className="hover-elevate active-elevate-2 overflow-visible h-full">
              <CardContent className="p-5 flex flex-col items-center gap-2 text-center">
                <Users className="w-8 h-8 text-blue-500" />
                <p className="text-sm font-medium text-card-foreground">Xodimlar</p>
              </CardContent>
            </Card>
          </button>
          <button onClick={() => navigate("/groups")} className="focus:outline-none focus:ring-2 focus:ring-ring rounded-md">
            <Card className="hover-elevate active-elevate-2 overflow-visible h-full">
              <CardContent className="p-5 flex flex-col items-center gap-2 text-center">
                <FolderOpen className="w-8 h-8 text-emerald-500" />
                <p className="text-sm font-medium text-card-foreground">Guruhlar</p>
              </CardContent>
            </Card>
          </button>
          <button onClick={() => navigate("/attendance")} className="focus:outline-none focus:ring-2 focus:ring-ring rounded-md">
            <Card className="hover-elevate active-elevate-2 overflow-visible h-full">
              <CardContent className="p-5 flex flex-col items-center gap-2 text-center">
                <CalendarCheck className="w-8 h-8 text-violet-500" />
                <p className="text-sm font-medium text-card-foreground">Davomat</p>
              </CardContent>
            </Card>
          </button>
        </div>
      </main>
    </div>
  );
}
