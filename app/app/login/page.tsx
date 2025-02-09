"use client";
import {login} from "@/services/ApiService";
import { useRouter } from 'next/navigation'


export default function Login() {
    const router = useRouter()

    const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
        event.preventDefault();
        const form = event.currentTarget;
        const username = form.username.value;
        const password = form.password.value;
        login(username, password).then(token => {
            localStorage.setItem("token", token);
            router.push("/");
        }).catch(error => {
            alert(error.message);
        });
    }

    return (
        <div className="h-screen flex flex-col items-center justify-center">
            <h1 className="text-center">Login</h1>
            <form
                className="flex flex-col gap-4 mx-auto mt-8 w-[90%] lg:w-1/4"
                onSubmit={handleSubmit}>
                <label htmlFor="username">Username</label>
                <input type="text" id="username" name="username"/>
                <label htmlFor="password">Password</label>
                <input type="password" id="password" name="password"/>
                <button type="submit" className="primary">Login</button>
            </form>
        </div>
    );
}
