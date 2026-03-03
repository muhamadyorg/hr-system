import { useState } from "react";
import { useQuery, useMutation } from "@tanstack/react-query";
import { useLocation } from "wouter";
import { useAuth } from "@/lib/auth";
import { apiRequest, queryClient } from "@/lib/queryClient";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from "@/components/ui/alert-dialog";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useToast } from "@/hooks/use-toast";
import { ArrowLeft, ShieldCheck, UserPlus, FolderPlus, Users, Trash2, Loader2, Eye, EyeOff } from "lucide-react";
import type { User, Employee, Group } from "@shared/schema";

interface AdminUser extends User {
  groupCount?: number;
}

export default function SudoPage() {
  const [, navigate] = useLocation();
  const { user } = useAuth();
  const { toast } = useToast();

  const [showAddAdmin, setShowAddAdmin] = useState(false);
  const [newAdmin, setNewAdmin] = useState({ username: "", password: "", fullName: "" });
  const [showNewPassword, setShowNewPassword] = useState(false);

  if (user?.role !== "sudo") {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <div className="text-center">
          <ShieldCheck className="w-12 h-12 text-destructive mx-auto mb-3" />
          <p className="text-foreground font-medium">Ruxsat yo'q</p>
          <p className="text-sm text-muted-foreground mt-1">Bu sahifa faqat Sudo uchun</p>
          <Button className="mt-4" onClick={() => navigate("/")}>Bosh sahifaga</Button>
        </div>
      </div>
    );
  }

  const { data: admins = [], isLoading: adminsLoading } = useQuery<AdminUser[]>({
    queryKey: ["/api/admins"],
  });

  const { data: employees = [], isLoading: empsLoading } = useQuery<(Employee & { groupName?: string })[]>({
    queryKey: ["/api/employees"],
  });

  const { data: groups = [] } = useQuery<(Group & { employeeCount: number })[]>({
    queryKey: ["/api/groups"],
  });

  const addAdminMutation = useMutation({
    mutationFn: async (data: any) => {
      const res = await apiRequest("POST", "/api/admins", data);
      return res.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["/api/admins"] });
      setShowAddAdmin(false);
      setNewAdmin({ username: "", password: "", fullName: "" });
      toast({ title: "Muvaffaqiyatli", description: "Admin qo'shildi" });
    },
    onError: (err: any) => {
      toast({ title: "Xatolik", description: err.message, variant: "destructive" });
    },
  });

  const deleteAdminMutation = useMutation({
    mutationFn: async (id: number) => {
      await apiRequest("DELETE", `/api/admins/${id}`);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["/api/admins"] });
      toast({ title: "Muvaffaqiyatli", description: "Admin o'chirildi" });
    },
    onError: (err: any) => {
      toast({ title: "Xatolik", description: err.message, variant: "destructive" });
    },
  });

  const handleAddAdmin = () => {
    if (!newAdmin.username.trim() || !newAdmin.password.trim() || !newAdmin.fullName.trim()) {
      toast({ title: "Xatolik", description: "Barcha maydonlarni to'ldiring", variant: "destructive" });
      return;
    }
    addAdminMutation.mutate(newAdmin);
  };

  return (
    <div className="min-h-screen bg-background">
      <header className="border-b border-border sticky top-0 z-50 bg-background">
        <div className="max-w-7xl mx-auto px-4 py-3 flex items-center justify-between gap-2 flex-wrap">
          <div className="flex items-center gap-2">
            <Button variant="ghost" size="icon" onClick={() => navigate("/")} data-testid="button-back">
              <ArrowLeft className="w-4 h-4" />
            </Button>
            <div className="flex items-center gap-2">
              <ShieldCheck className="w-5 h-5 text-primary" />
              <h1 className="text-lg font-bold text-foreground">Sudo Panel</h1>
            </div>
          </div>
          <div className="flex items-center gap-2 flex-wrap">
            <Button onClick={() => setShowAddAdmin(true)} data-testid="button-add-admin">
              <UserPlus className="w-4 h-4 mr-1" /> Admin qo'shish
            </Button>
            <Button variant="secondary" onClick={() => navigate("/groups")} data-testid="button-go-groups">
              <FolderPlus className="w-4 h-4 mr-1" /> Guruh yaratish
            </Button>
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-4 py-6 space-y-6">
        <Tabs defaultValue="employees" className="w-full">
          <TabsList className="w-full max-w-md">
            <TabsTrigger value="employees" className="flex-1" data-testid="tab-employees">Xodimlar</TabsTrigger>
            <TabsTrigger value="admins" className="flex-1" data-testid="tab-admins">Adminlar</TabsTrigger>
            <TabsTrigger value="groups" className="flex-1" data-testid="tab-groups">Guruhlar</TabsTrigger>
          </TabsList>

          <TabsContent value="employees" className="mt-4 space-y-3">
            <h3 className="text-base font-semibold text-foreground">
              Barcha xodimlar ({employees.length})
            </h3>
            {empsLoading ? (
              <div className="space-y-2">
                {[1,2,3,4,5].map(i => <Skeleton key={i} className="h-14 rounded-md" />)}
              </div>
            ) : employees.length === 0 ? (
              <p className="text-sm text-muted-foreground text-center py-8">Xodimlar topilmadi</p>
            ) : (
              <div className="space-y-2">
                {employees.map((emp) => (
                  <Card
                    key={emp.id}
                    className="hover-elevate cursor-pointer overflow-visible"
                    onClick={() => navigate(`/employees/${emp.id}`)}
                    data-testid={`card-sudo-employee-${emp.id}`}
                  >
                    <CardContent className="p-3 flex items-center gap-3">
                      <Avatar className="w-9 h-9">
                        <AvatarFallback className="bg-primary/10 text-primary text-sm font-medium">
                          {emp.fullName.split(" ").map(n => n[0]).join("").slice(0, 2).toUpperCase()}
                        </AvatarFallback>
                      </Avatar>
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-medium text-card-foreground truncate">{emp.fullName}</p>
                        <p className="text-xs text-muted-foreground truncate">
                          #{emp.employeeNo} · {(emp as any).groupName ?? "Guruhsiz"}
                        </p>
                      </div>
                      <span className="text-xs text-muted-foreground">{emp.position ?? ""}</span>
                    </CardContent>
                  </Card>
                ))}
              </div>
            )}
          </TabsContent>

          <TabsContent value="admins" className="mt-4 space-y-3">
            <h3 className="text-base font-semibold text-foreground">
              Adminlar ({admins.length})
            </h3>
            {adminsLoading ? (
              <div className="space-y-2">
                {[1,2,3].map(i => <Skeleton key={i} className="h-14 rounded-md" />)}
              </div>
            ) : admins.length === 0 ? (
              <p className="text-sm text-muted-foreground text-center py-8">Adminlar topilmadi</p>
            ) : (
              <div className="space-y-2">
                {admins.map((admin) => (
                  <Card key={admin.id} className="overflow-visible" data-testid={`card-admin-${admin.id}`}>
                    <CardContent className="p-3 flex items-center justify-between gap-3">
                      <div className="flex items-center gap-3 min-w-0">
                        <Avatar className="w-9 h-9">
                          <AvatarFallback className="bg-primary/10 text-primary text-sm font-medium">
                            {admin.fullName.split(" ").map(n => n[0]).join("").slice(0, 2).toUpperCase()}
                          </AvatarFallback>
                        </Avatar>
                        <div className="min-w-0">
                          <p className="text-sm font-medium text-card-foreground truncate">{admin.fullName}</p>
                          <p className="text-xs text-muted-foreground">@{admin.username}</p>
                        </div>
                      </div>
                      <div className="flex items-center gap-2 flex-shrink-0">
                        <Badge variant="secondary" className="text-xs">
                          {admin.role === "sudo" ? "SUDO" : "ADMIN"}
                        </Badge>
                        {admin.role !== "sudo" && (
                          <AlertDialog>
                            <AlertDialogTrigger asChild>
                              <Button variant="ghost" size="icon" data-testid={`button-delete-admin-${admin.id}`}>
                                <Trash2 className="w-4 h-4 text-destructive" />
                              </Button>
                            </AlertDialogTrigger>
                            <AlertDialogContent>
                              <AlertDialogHeader>
                                <AlertDialogTitle>Adminni o'chirish</AlertDialogTitle>
                                <AlertDialogDescription>
                                  "{admin.fullName}" admin hisobini o'chirmoqchimisiz?
                                </AlertDialogDescription>
                              </AlertDialogHeader>
                              <AlertDialogFooter>
                                <AlertDialogCancel>Bekor qilish</AlertDialogCancel>
                                <AlertDialogAction onClick={() => deleteAdminMutation.mutate(admin.id)}>
                                  Ha, o'chirish
                                </AlertDialogAction>
                              </AlertDialogFooter>
                            </AlertDialogContent>
                          </AlertDialog>
                        )}
                      </div>
                    </CardContent>
                  </Card>
                ))}
              </div>
            )}
          </TabsContent>

          <TabsContent value="groups" className="mt-4 space-y-3">
            <div className="flex items-center justify-between gap-2 flex-wrap">
              <h3 className="text-base font-semibold text-foreground">
                Guruhlar ({groups.length})
              </h3>
              <Button size="sm" onClick={() => navigate("/groups")} data-testid="button-manage-groups">
                Boshqarish
              </Button>
            </div>
            {groups.length === 0 ? (
              <p className="text-sm text-muted-foreground text-center py-8">Guruhlar topilmadi</p>
            ) : (
              <div className="space-y-2">
                {groups.map((g) => (
                  <Card key={g.id} className="overflow-visible" data-testid={`card-sudo-group-${g.id}`}>
                    <CardContent className="p-3 flex items-center justify-between gap-3">
                      <div className="min-w-0">
                        <p className="text-sm font-medium text-card-foreground">{g.name}</p>
                        <p className="text-xs text-muted-foreground">Login: {g.login} · Parol: {g.password}</p>
                      </div>
                      <Badge variant="secondary" className="text-xs flex-shrink-0">
                        <Users className="w-3 h-3 mr-1" />{(g as any).employeeCount ?? 0}
                      </Badge>
                    </CardContent>
                  </Card>
                ))}
              </div>
            )}
          </TabsContent>
        </Tabs>
      </main>

      <Dialog open={showAddAdmin} onOpenChange={setShowAddAdmin}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Yangi admin qo'shish</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>To'liq ismi *</Label>
              <Input
                value={newAdmin.fullName}
                onChange={e => setNewAdmin(p => ({ ...p, fullName: e.target.value }))}
                placeholder="Admin ismi"
                data-testid="input-admin-fullname"
              />
            </div>
            <div className="space-y-2">
              <Label>Foydalanuvchi nomi *</Label>
              <Input
                value={newAdmin.username}
                onChange={e => setNewAdmin(p => ({ ...p, username: e.target.value }))}
                placeholder="Login"
                data-testid="input-admin-username"
              />
            </div>
            <div className="space-y-2">
              <Label>Parol *</Label>
              <div className="relative">
                <Input
                  value={newAdmin.password}
                  onChange={e => setNewAdmin(p => ({ ...p, password: e.target.value }))}
                  placeholder="Parol"
                  type={showNewPassword ? "text" : "password"}
                  className="pr-10"
                  data-testid="input-admin-password"
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="absolute right-0 top-0 h-full"
                  onClick={() => setShowNewPassword(!showNewPassword)}
                >
                  {showNewPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                </Button>
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button variant="secondary" onClick={() => setShowAddAdmin(false)}>Bekor qilish</Button>
            <Button onClick={handleAddAdmin} disabled={addAdminMutation.isPending} data-testid="button-confirm-add-admin">
              {addAdminMutation.isPending && <Loader2 className="w-4 h-4 animate-spin mr-1" />}
              Qo'shish
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
