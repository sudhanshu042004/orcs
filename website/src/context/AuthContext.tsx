import { createContext, type ReactNode, useEffect, useState } from "react";
import { apiUrl } from "../utils/contants";

interface AuthContextType {
    user: User | null;
    loading: boolean;
    setUser: (user: User | null) => void;
    logout: () => void;
}

export const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider = ({ children }: { children: ReactNode }) => {
    const [user, setUser] = useState<User | null>(null);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        async function fetchUser() {
            try {
                const res = await fetch(apiUrl + "api/user",{
                    method : "GET",
                    credentials : "include"
                });
                if (res.ok) {
                    const data = await res.json();
                    setUser(data.data);
                } else {
                    setUser(null);
                }
            } catch (error) {
                console.error("Auth fetch failed:", error);
                setUser(null);
            } finally {
                setLoading(false);
            }
        }
        fetchUser();
    }, []);

    const logout = () => {
        setUser(null);
        // Add backend logout call
    };

    return (
        <AuthContext.Provider value={{ user, loading, setUser, logout }}>
            {children}
        </AuthContext.Provider>
    );
};