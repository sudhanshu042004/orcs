import { Link } from 'react-router-dom'

const Navbar = () => {
  return (
    <div className='relative flex justify-between font-medium pb-2 bg-black text-white'>
        <div>orcs</div>
        <div>
            <Link to='/login'>
                SignIn
            </Link>
        </div>

        <div className="absolute bottom-0 left-0 w-full h-px bg-linear-to-r from-transparent via-white/80 to-transparent" />
    </div>
  )
}

export default Navbar