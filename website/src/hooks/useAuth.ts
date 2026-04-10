import { useEffect, useState } from "react";
import { useUser } from "./useUser"
import { apiUrl } from "../utils/contants";

export const useAuth = () =>{
    const {user,addUser,removeUser,setUser} = useUser();
    const [loading,setLoading] = useState(true);
    
    useEffect(()=>{
        // fetching user data
        async function fetching(){
            try {
                const res = await fetch(apiUrl + '/api/user');
                const data = await res.json();
                setUser(data)
            } catch (error) {
                
            } finally {
                setLoading(false)
            }
        }
        fetching();
    },[addUser]);

    const logout = () =>{
        removeUser();
        //need to implement backend for this also
    }
    return {user,logout,loading,setUser};
}