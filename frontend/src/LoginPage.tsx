import React, { useEffect } from 'react';
import { useRecoilState } from 'recoil';
import { Link } from 'react-router-dom';

import { idTokenState } from './state';

declare global {
    interface Window { google: any; }
}

export default function LoginPage() {
    const [, setIdToken] = useRecoilState(idTokenState);

    useEffect(() => {
        function handleCredentialResponse(response: any) {
            console.log(response.credential);
            setIdToken(response.credential);
        }

        window.google.accounts.id.initialize({
            client_id: '691474794551-sf5s8aprb3dnus95ic78048l2497ornp.apps.googleusercontent.com',
            callback: handleCredentialResponse,
            auto_select: true
        });

        window.google.accounts.id.prompt();

        window.google.accounts.id.renderButton(document.getElementById("buttonDiv"), {
            theme: 'outline',
            size: 'large',
        });
    }, [setIdToken]);

    return (
        <div>
            <div id="buttonDiv"></div>
            LoginPage
            <div>
                <Link to="/home">Home</Link>
            </div>
        </div>
    )
}