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
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from "@/components/ui/alert-dialog";
import { useToast } from "@/hooks/use-toast";
import { ArrowLeft, Plus, FolderOpen, Users, Trash2, Loader2, Eye, EyeOff } from "lucide-react";
import type { Group, Employee } from "@shared/schema";

interface GroupWithCount extends Group {
  employeeCount: number;
}

export default function GroupsPage() {
  const [, navigate] = useLocation();
  const { user } = useAuth();
  const { toast } = useToast();
  const isSudo = user?.role === "sudo";
  const [showCreate, setShowCreate] = useState(false);
  const [newGroup, setNewGroup] = useState({ name: "", login: "", password: "", description: "" });
  const [showPasswords, setShowPasswords] = useState<Record<number, boolean>>({});
  const [expandedGroup, setExpandedGroup] = useState<number | null>(null);

  const { data: groups = [], isLoading } = useQuery<GroupWithCount[]>({
    queryKey: ["/api/groups"],
  });

  const { data: groupEmployees = [] } = useQuery<Employee[]>({
    queryKey: ["/api/groups", expandedGroup, "employees"],
    enabled: expandedGroup !== null,
  });

  const createMutation = useMutation({
    mutationFn: async (data: any) => {
      const res = await apiRequest("POST", "/api/groups", data);
      return res.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["/api/groups"] });
      queryClient.invalidateQueries({ queryKey: ["/api/dashboard/stats"] });
      setShowCreate(false);
      setNewGroup({ name: "", login: "", password: "", description: "" });
      toast({ title: "Muvaffaqiyatli", description: "Guruh yaratildi" });
    },
    onError: (err: any) => {
      toast({ title: "Xatolik", description: err.message, variant: "destructive" });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async (id: number) => {
      await apiRequest("DELETE", `/api/groups/${id}`);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["/api/groups"] });
      queryClient.invalidateQueries({ queryKey: ["/api/dashboard/stats"] });
      toast({ title: "Muvaffaqiyatli", description: "Guruh o'chirildi" });
    },
    onError: (err: any) => {
      toast({ title: "Xatolik", description: err.message, variant: "destructive" });
    },
  });

  const handleCreate = () => {
    if (!newGroup.name.trim() || !newGroup.login.trim() || !newGroup.password.trim()) {
      toast({ title: "Xatolik", description: "Guruh nomi, login va parol kiritilishi shart", variant: "destructive" });
      return;
    }
    createMutation.mutate({
      name: newGroup.name,
      login: newGroup.login,
      password: newGroup.password,
      description: newGroup.description || null,
      createdBy: user!.id,
    });
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
              <FolderOpen className="w-5 h-5 text-primary" />
              <h1 className="text-lg font-bold text-foreground">Guruhlar</h1>
            </div>
          </div>
          {isSudo && (
            <Button onClick={() => setShowCreate(true)} data-testid="button-create-group">
              <Plus className="w-4 h-4 mr-1" /> Guruh yaratish
            </Button>
          )}
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-4 py-6 space-y-4">
        <Badge variant="secondary">{groups.length} ta guruh</Badge>

        {isLoading ? (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {[1,2,3,4].map(i => <Skeleton key={i} className="h-32 rounded-md" />)}
          </div>
        ) : groups.length === 0 ? (
          <div className="text-center py-16">
            <FolderOpen className="w-12 h-12 text-muted-foreground mx-auto mb-3" />
            <p className="text-muted-foreground">Hali guruhlar yaratilmagan</p>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {groups.map((group) => (
              <Card key={group.id} className="overflow-visible" data-testid={`card-group-${group.id}`}>
                <CardContent className="p-5 space-y-3">
                  <div className="flex items-start justify-between gap-2">
                    <div className="min-w-0">
                      <h3 className="text-base font-semibold text-card-foreground">{group.name}</h3>
                      {group.description && (
                        <p className="text-xs text-muted-foreground mt-0.5">{group.description}</p>
                      )}
                    </div>
                    <div className="flex items-center gap-1 flex-shrink-0">
                      <Badge variant="secondary" className="text-xs">
                        <Users className="w-3 h-3 mr-1" /> {group.employeeCount}
                      </Badge>
                      {isSudo && (
                        <AlertDialog>
                          <AlertDialogTrigger asChild>
                            <Button variant="ghost" size="icon" data-testid={`button-delete-group-${group.id}`}>
                              <Trash2 className="w-4 h-4 text-destructive" />
                            </Button>
                          </AlertDialogTrigger>
                          <AlertDialogContent>
                            <AlertDialogHeader>
                              <AlertDialogTitle>Guruhni o'chirish</AlertDialogTitle>
                              <AlertDialogDescription>
                                "{group.name}" guruhini o'chirmoqchimisiz? Guruh ichidagi xodimlar guruhsiz qoladi.
                              </AlertDialogDescription>
                            </AlertDialogHeader>
                            <AlertDialogFooter>
                              <AlertDialogCancel>Bekor qilish</AlertDialogCancel>
                              <AlertDialogAction onClick={() => deleteMutation.mutate(group.id)}>
                                Ha, o'chirish
                              </AlertDialogAction>
                            </AlertDialogFooter>
                          </AlertDialogContent>
                        </AlertDialog>
                      )}
                    </div>
                  </div>

                  {isSudo && (
                    <div className="p-3 rounded-md bg-muted/30 space-y-1">
                      <div className="flex items-center justify-between gap-2">
                        <span className="text-xs text-muted-foreground">Login:</span>
                        <span className="text-xs font-mono text-foreground">{group.login}</span>
                      </div>
                      <div className="flex items-center justify-between gap-2">
                        <span className="text-xs text-muted-foreground">Parol:</span>
                        <div className="flex items-center gap-1">
                          <span className="text-xs font-mono text-foreground">
                            {showPasswords[group.id] ? group.password : "••••••"}
                          </span>
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-6 w-6"
                            onClick={() => setShowPasswords(p => ({ ...p, [group.id]: !p[group.id] }))}
                          >
                            {showPasswords[group.id] ? <EyeOff className="w-3 h-3" /> : <Eye className="w-3 h-3" />}
                          </Button>
                        </div>
                      </div>
                    </div>
                  )}

                  <Button
                    variant="secondary"
                    size="sm"
                    className="w-full"
                    onClick={() => setExpandedGroup(expandedGroup === group.id ? null : group.id)}
                    data-testid={`button-expand-group-${group.id}`}
                  >
                    <Users className="w-4 h-4 mr-1" />
                    Xodimlarni ko'rish
                  </Button>

                  {expandedGroup === group.id && (
                    <div className="space-y-1 pt-1">
                      {groupEmployees.length === 0 ? (
                        <p className="text-xs text-muted-foreground text-center py-3">Bu guruhda xodim yo'q</p>
                      ) : (
                        groupEmployees.map((emp) => (
                          <div
                            key={emp.id}
                            className="flex items-center gap-2 p-2 rounded-md hover-elevate cursor-pointer"
                            onClick={() => navigate(`/employees/${emp.id}`)}
                          >
                            <span className="text-sm text-foreground truncate">{emp.fullName}</span>
                            <span className="text-xs text-muted-foreground">#{emp.employeeNo}</span>
                          </div>
                        ))
                      )}
                    </div>
                  )}
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </main>

      <Dialog open={showCreate} onOpenChange={setShowCreate}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Yangi guruh yaratish</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>Guruh nomi *</Label>
              <Input
                value={newGroup.name}
                onChange={e => setNewGroup(p => ({ ...p, name: e.target.value }))}
                placeholder="Masalan: IT bo'limi"
                data-testid="input-group-name"
              />
            </div>
            <div className="space-y-2">
              <Label>Login *</Label>
              <Input
                value={newGroup.login}
                onChange={e => setNewGroup(p => ({ ...p, login: e.target.value }))}
                placeholder="Guruh login"
                data-testid="input-group-login"
              />
            </div>
            <div className="space-y-2">
              <Label>Parol *</Label>
              <Input
                value={newGroup.password}
                onChange={e => setNewGroup(p => ({ ...p, password: e.target.value }))}
                placeholder="Guruh paroli"
                type="password"
                data-testid="input-group-password"
              />
            </div>
            <div className="space-y-2">
              <Label>Tavsif</Label>
              <Input
                value={newGroup.description}
                onChange={e => setNewGroup(p => ({ ...p, description: e.target.value }))}
                placeholder="Guruh haqida qisqacha"
                data-testid="input-group-description"
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="secondary" onClick={() => setShowCreate(false)}>Bekor qilish</Button>
            <Button onClick={handleCreate} disabled={createMutation.isPending} data-testid="button-confirm-create-group">
              {createMutation.isPending && <Loader2 className="w-4 h-4 animate-spin mr-1" />}
              Yaratish
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
